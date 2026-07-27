package main

import (
	"csu-star-backend/config"
	"csu-star-backend/internal/docengine"
	"csu-star-backend/logger"
	"csu-star-backend/pkg/utils"
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	logger.Init()
	if err := config.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cfg := config.GetConfig()
	utils.InitSnowflake(cfg.Snowflake.NodeID)
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	store := docengine.NewGormStore(db)
	if err := store.AutoMigrate(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate", err)
		os.Exit(1)
	}
	n, err := docengine.RepairAndImportPublishedWiki(db)
	if err != nil {
		fmt.Fprintln(os.Stderr, "repair", err)
		os.Exit(1)
	}
	fmt.Println("repaired+imported pages:", n)

	type sample struct {
		Title       string
		ParentTitle *string
		ParentSpace *string
	}
	var samples []sample
	db.Raw(`
SELECT p.title, pp.title as parent_title, pp.space_key as parent_space
FROM compass_pages p
LEFT JOIN compass_pages pp ON pp.id = p.parent_id
WHERE p.space_key = 'majors' AND p.parent_id IS NOT NULL AND p.deleted_at IS NULL
ORDER BY p.title LIMIT 8
`).Scan(&samples)
	for _, s := range samples {
		pt, ps := "<nil>", "<nil>"
		if s.ParentTitle != nil {
			pt = *s.ParentTitle
		}
		if s.ParentSpace != nil {
			ps = *s.ParentSpace
		}
		fmt.Printf("  %s → parent=%s (%s)\n", s.Title, pt, ps)
	}
	var counts []struct {
		SpaceKey string
		C        int64
	}
	db.Raw(`SELECT space_key, count(*) as c FROM compass_pages WHERE deleted_at IS NULL GROUP BY space_key ORDER BY 1`).Scan(&counts)
	fmt.Println("counts:", counts)
}
