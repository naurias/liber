package main

import "fmt"

func runDelete(id int, rest []string) error {
	yes := false
	for _, a := range rest {
		switch a {
		case "-y", "--yes":
			yes = true
		default:
			return fmt.Errorf("unknown flag for -d: %s", a)
		}
	}

	cfg, store, err := loadCfgAndStore()
	if err != nil {
		return err
	}
	b := store.Find(id)
	if b == nil {
		return fmt.Errorf("no bookmark with id %d (see `liber -l`)", id)
	}

	if !yes && !confirm(fmt.Sprintf("Delete [%d] %s?", b.ID, b.Title), false) {
		fmt.Println("Cancelled.")
		return nil
	}

	deleteBookmarkFiles(cfg, b)
	store.Delete(b.ID)
	if err := store.Save(); err != nil {
		return fmt.Errorf("saving index: %w", err)
	}
	fmt.Printf("Deleted [%d] %s\n", b.ID, b.Title)
	return nil
}
