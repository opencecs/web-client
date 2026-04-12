package main

import (
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"
)

type ContainerAliasService struct {
	db       *sql.DB
	migrated sync.Once
}

func NewContainerAliasService(db *sql.DB) *ContainerAliasService {
	s := &ContainerAliasService{db: db}
	s.initTable()
	return s
}

func (s *ContainerAliasService) initTable() {
	// 新表：以坑位号为主键（刷机/重置后容器名会变，但坑位号不变）
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS slot_aliases (
			slot_num INTEGER PRIMARY KEY,
			alias TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal("[Alias] 创建 slot_aliases 表失败:", err)
	}
}

// MigrateFromNameBased 从旧的 name-based 表迁移到 slot-based 表（只执行一次）
func (s *ContainerAliasService) MigrateFromNameBased(containers []ParsedContainer) {
	s.migrated.Do(func() {
		// 检查旧表是否存在
		var tableName string
		err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='container_aliases'").Scan(&tableName)
		if err != nil {
			return // 旧表不存在，无需迁移
		}

		// 读取旧别名
		rows, err := s.db.Query("SELECT name, alias FROM container_aliases")
		if err != nil {
			return
		}
		defer rows.Close()

		// 构建容器名 → 坑位号映射
		nameToSlot := make(map[string]int, len(containers))
		for _, c := range containers {
			nameToSlot[c.Name] = c.IndexNum
		}

		migrated := 0
		for rows.Next() {
			var name, alias string
			if rows.Scan(&name, &alias) != nil {
				continue
			}
			slot, ok := nameToSlot[name]
			if !ok || slot <= 0 {
				continue
			}
			// 仅迁移新表中不存在的坑位
			var exists int
			s.db.QueryRow("SELECT COUNT(*) FROM slot_aliases WHERE slot_num = ?", slot).Scan(&exists)
			if exists == 0 {
				s.db.Exec("INSERT INTO slot_aliases (slot_num, alias, updated_at) VALUES (?, ?, ?)",
					slot, alias, time.Now())
				migrated++
			}
		}

		if migrated > 0 {
			log.Printf("[Alias] 从旧表迁移了 %d 条别名到坑位模式", migrated)
		}

		// 迁移完成后删除旧表
		s.db.Exec("DROP TABLE IF EXISTS container_aliases")
	})
}

// SetAlias 设置坑位别名
func (s *ContainerAliasService) SetAlias(slotNum int, alias string) error {
	alias = strings.TrimSpace(alias)
	if slotNum <= 0 || alias == "" {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO slot_aliases (slot_num, alias, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(slot_num) DO UPDATE SET alias = excluded.alias, updated_at = excluded.updated_at`,
		slotNum, alias, time.Now(),
	)
	return err
}

// DeleteAlias 删除坑位别名
func (s *ContainerAliasService) DeleteAlias(slotNum int) error {
	if slotNum <= 0 {
		return nil
	}
	_, err := s.db.Exec("DELETE FROM slot_aliases WHERE slot_num = ?", slotNum)
	return err
}

// GetSlotAliases 返回 slot_num → alias 映射
func (s *ContainerAliasService) GetSlotAliases() map[int]string {
	aliases := make(map[int]string)
	rows, err := s.db.Query("SELECT slot_num, alias FROM slot_aliases")
	if err != nil {
		log.Printf("[Alias] 查询失败: %v", err)
		return aliases
	}
	defer rows.Close()
	for rows.Next() {
		var slot int
		var alias string
		if rows.Scan(&slot, &alias) == nil {
			aliases[slot] = alias
		}
	}
	return aliases
}

// BuildAliasMap 构建 containerName → alias 映射（前端兼容格式）
// 根据当前容器列表，将坑位别名映射到对应的容器名
func (s *ContainerAliasService) BuildAliasMap(containers []ParsedContainer) map[string]string {
	slotAliases := s.GetSlotAliases()
	if len(slotAliases) == 0 {
		return map[string]string{}
	}

	result := make(map[string]string, len(slotAliases))
	for _, c := range containers {
		if alias, ok := slotAliases[c.IndexNum]; ok {
			result[c.Name] = alias
		}
	}
	return result
}
