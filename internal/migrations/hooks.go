package migrations

import "gorm.io/gorm"

type MigrationHook interface {
	BeforeMigrate(db *gorm.DB) error
	AfterMigrate(db *gorm.DB) error
	BeforeRollback(db *gorm.DB) error
	AfterRollback(db *gorm.DB) error
}

var registeredHooks []MigrationHook

func RegisterHook(h MigrationHook) {
	if h == nil {
		return
	}
	registeredHooks = append(registeredHooks, h)
}

func fireBeforeMigrate(db *gorm.DB) error {
	for _, h := range registeredHooks {
		if err := h.BeforeMigrate(db); err != nil {
			return err
		}
	}
	return nil
}

func fireAfterMigrate(db *gorm.DB) error {
	for _, h := range registeredHooks {
		if err := h.AfterMigrate(db); err != nil {
			return err
		}
	}
	return nil
}

func fireBeforeRollback(db *gorm.DB) error {
	for _, h := range registeredHooks {
		if err := h.BeforeRollback(db); err != nil {
			return err
		}
	}
	return nil
}

func fireAfterRollback(db *gorm.DB) error {
	for _, h := range registeredHooks {
		if err := h.AfterRollback(db); err != nil {
			return err
		}
	}
	return nil
}
