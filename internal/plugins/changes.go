package plugins

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/snakepilot10/ozsh/internal/config"
)

// PendingAdd describes an inactive checkout that will be moved into its final
// managed location only when Review & Apply succeeds.
type PendingAdd struct {
	Name          string
	RepositoryURL string
	StagingDir    string
	FinalDir      string
	Load          string
}

// PendingRemove describes an installed checkout that will be quarantined only
// when Review & Apply begins.
type PendingRemove struct {
	Name   string
	Source string
}

// ChangeSet contains filesystem work associated with a pending config model.
type ChangeSet struct {
	Adds    []PendingAdd
	Removes []PendingRemove
}

// Clone returns an independent copy suitable for Review & Apply snapshots.
func (changes ChangeSet) Clone() ChangeSet {
	return ChangeSet{
		Adds:    append([]PendingAdd(nil), changes.Adds...),
		Removes: append([]PendingRemove(nil), changes.Removes...),
	}
}

func (changes ChangeSet) Empty() bool {
	return len(changes.Adds) == 0 && len(changes.Removes) == 0
}

func (changes ChangeSet) Counts() (adds, removes int) {
	return len(changes.Adds), len(changes.Removes)
}

func (changes ChangeSet) RepositoryURLFor(name string) (string, bool) {
	for _, addition := range changes.Adds {
		if addition.Name == name {
			return addition.RepositoryURL, true
		}
	}
	return "", false
}

// RootFor returns the checkout that currently contains a plugin's load file.
func (changes ChangeSet) RootFor(name, finalSource string) string {
	for _, addition := range changes.Adds {
		if addition.Name == name {
			return addition.StagingDir
		}
	}
	return finalSource
}

// QueueAdd records a trusted pending item without activating or moving it.
func (changes *ChangeSet) QueueAdd(cfg *config.Config, stage StagedRepository, load string) error {
	if changes == nil {
		return fmt.Errorf("plugin change set is nil")
	}
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := validateName(stage.Repository.Name); err != nil {
		return err
	}
	if err := validateManagedStaging(stage.StagingDir); err != nil {
		return err
	}
	if err := validateManagedFinal(stage.Repository.Name, stage.FinalDir); err != nil {
		return err
	}
	load = filepath.Clean(strings.TrimSpace(load))
	if err := ValidateCandidate(stage.StagingDir, load); err != nil {
		return fmt.Errorf("invalid plugin load file: %w", err)
	}
	if err := ValidateNewRepository(cfg, stage.Repository); err != nil {
		return err
	}
	for _, addition := range changes.Adds {
		if addition.Name == stage.Repository.Name || filepath.Clean(addition.FinalDir) == filepath.Clean(stage.FinalDir) {
			return fmt.Errorf("plugin %q already has a pending add", stage.Repository.Name)
		}
	}
	for _, removal := range changes.Removes {
		if removal.Name == stage.Repository.Name {
			return fmt.Errorf("plugin %q already has a pending removal", stage.Repository.Name)
		}
	}
	if _, err := os.Lstat(stage.FinalDir); err == nil {
		return fmt.Errorf("plugin destination already exists: %s", stage.FinalDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect plugin destination: %w", err)
	}

	load = filepath.ToSlash(load)
	changes.Adds = append(changes.Adds, PendingAdd{
		Name:          stage.Repository.Name,
		RepositoryURL: stage.Repository.URL,
		StagingDir:    filepath.Clean(stage.StagingDir),
		FinalDir:      filepath.Clean(stage.FinalDir),
		Load:          load,
	})
	cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{
		Name:    stage.Repository.Name,
		Enabled: true,
		Trusted: true,
		Source:  filepath.Clean(stage.FinalDir),
		Load:    load,
	})
	return nil
}

// QueueRemove removes an item from the pending config while leaving installed
// files untouched until Begin. Removing an unapplied add cancels and cleans it.
func (changes *ChangeSet) QueueRemove(cfg *config.Config, name string) error {
	if changes == nil {
		return fmt.Errorf("plugin change set is nil")
	}
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := validateName(name); err != nil {
		return err
	}

	for i, addition := range changes.Adds {
		if addition.Name != name {
			continue
		}
		if err := (StagedRepository{StagingDir: addition.StagingDir}).Cleanup(); err != nil {
			return fmt.Errorf("cancel pending plugin add: %w", err)
		}
		changes.Adds = append(changes.Adds[:i], changes.Adds[i+1:]...)
		removeConfigPlugin(cfg, name)
		return nil
	}

	for _, removal := range changes.Removes {
		if removal.Name == name {
			return fmt.Errorf("plugin %q already has a pending removal", name)
		}
	}
	index := -1
	var item config.PluginItem
	for i, candidate := range cfg.Plugins.Items {
		if candidate.Name == name {
			index = i
			item = candidate
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("plugin %q not found", name)
	}
	if err := validateManagedPluginSource(name, item.Source); err != nil {
		return fmt.Errorf("plugin %q has an unsafe source path: %w", name, err)
	}
	changes.Removes = append(changes.Removes, PendingRemove{Name: name, Source: filepath.Clean(item.Source)})
	cfg.Plugins.Items = append(cfg.Plugins.Items[:index], cfg.Plugins.Items[index+1:]...)
	return nil
}

// Cleanup removes every still-unapplied staging checkout.
func (changes ChangeSet) Cleanup() error {
	var cleanupErrors []error
	for _, addition := range changes.Adds {
		if err := (StagedRepository{StagingDir: addition.StagingDir}).Cleanup(); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("cleanup plugin %q: %w", addition.Name, err))
		}
	}
	return errors.Join(cleanupErrors...)
}

type renameEntry struct {
	from string
	to   string
	kind string
}

// Transaction journals completed renames so Apply can either finalize or
// restore plugin paths exactly.
type Transaction struct {
	entries   []renameEntry
	committed bool
}

// Begin validates all paths before performing reversible renames.
func (changes ChangeSet) Begin(cfg *config.Config) (*Transaction, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if err := changes.validateAgainstConfig(cfg); err != nil {
		return nil, err
	}

	transaction := &Transaction{}
	rollbackOnError := func(primary error) (*Transaction, error) {
		rollbackErr := rollbackEntries(transaction.entries)
		return nil, errors.Join(primary, rollbackErr)
	}

	for _, addition := range changes.Adds {
		if err := os.Rename(addition.StagingDir, addition.FinalDir); err != nil {
			return rollbackOnError(fmt.Errorf("finalize plugin %q: %w", addition.Name, err))
		}
		transaction.entries = append(transaction.entries, renameEntry{
			from: addition.StagingDir,
			to:   addition.FinalDir,
			kind: "add",
		})
	}

	now := time.Now().UnixNano()
	for index, removal := range changes.Removes {
		quarantine := fmt.Sprintf("%s.ozsh-remove-%d-%d", removal.Source, now, index)
		if _, err := os.Lstat(quarantine); err == nil {
			return rollbackOnError(fmt.Errorf("plugin removal quarantine already exists: %s", quarantine))
		} else if !os.IsNotExist(err) {
			return rollbackOnError(fmt.Errorf("inspect plugin removal quarantine: %w", err))
		}
		if err := os.Rename(removal.Source, quarantine); err != nil {
			return rollbackOnError(fmt.Errorf("stage plugin removal %q: %w", removal.Name, err))
		}
		transaction.entries = append(transaction.entries, renameEntry{
			from: removal.Source,
			to:   quarantine,
			kind: "remove",
		})
	}
	return transaction, nil
}

func (changes ChangeSet) validateAgainstConfig(cfg *config.Config) error {
	seen := make(map[string]string, len(changes.Adds)+len(changes.Removes))
	items := make(map[string]config.PluginItem, len(cfg.Plugins.Items))
	for _, item := range cfg.Plugins.Items {
		items[item.Name] = item
	}

	for _, addition := range changes.Adds {
		if previous, exists := seen[addition.Name]; exists {
			return fmt.Errorf("plugin %q has conflicting pending %s and add", addition.Name, previous)
		}
		seen[addition.Name] = "add"
		if err := validateName(addition.Name); err != nil {
			return err
		}
		if err := validateManagedStaging(addition.StagingDir); err != nil {
			return fmt.Errorf("plugin %q staging path: %w", addition.Name, err)
		}
		if err := validateManagedFinal(addition.Name, addition.FinalDir); err != nil {
			return fmt.Errorf("plugin %q final path: %w", addition.Name, err)
		}
		if err := ValidateCandidate(addition.StagingDir, filepath.FromSlash(addition.Load)); err != nil {
			return fmt.Errorf("plugin %q load file: %w", addition.Name, err)
		}
		if _, err := os.Lstat(addition.FinalDir); err == nil {
			return fmt.Errorf("plugin %q final path already exists", addition.Name)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect plugin %q final path: %w", addition.Name, err)
		}
		item, exists := items[addition.Name]
		if !exists {
			return fmt.Errorf("pending config is missing plugin %q", addition.Name)
		}
		if filepath.Clean(item.Source) != filepath.Clean(addition.FinalDir) || filepath.ToSlash(filepath.Clean(item.Load)) != filepath.ToSlash(filepath.Clean(addition.Load)) {
			return fmt.Errorf("pending config does not match plugin %q add", addition.Name)
		}
		if !item.Trusted || !item.Enabled {
			return fmt.Errorf("pending plugin %q must be trusted and enabled", addition.Name)
		}
	}

	for _, removal := range changes.Removes {
		if previous, exists := seen[removal.Name]; exists {
			return fmt.Errorf("plugin %q has conflicting pending %s and removal", removal.Name, previous)
		}
		seen[removal.Name] = "removal"
		if err := validateName(removal.Name); err != nil {
			return err
		}
		if err := validateManagedPluginSource(removal.Name, removal.Source); err != nil {
			return fmt.Errorf("plugin %q removal source: %w", removal.Name, err)
		}
		if _, exists := items[removal.Name]; exists {
			return fmt.Errorf("pending config still contains removed plugin %q", removal.Name)
		}
	}
	return nil
}

// Commit deletes removal quarantines and leaves additions in final locations.
func (transaction *Transaction) Commit() error {
	if transaction == nil {
		return fmt.Errorf("plugin transaction is nil")
	}
	if transaction.committed {
		return fmt.Errorf("transaction already completed")
	}
	transaction.committed = true
	var commitErrors []error
	for _, entry := range transaction.entries {
		if entry.kind != "remove" {
			continue
		}
		if err := os.RemoveAll(entry.to); err != nil {
			commitErrors = append(commitErrors, fmt.Errorf("remove plugin quarantine %s: %w", entry.to, err))
		}
	}
	return errors.Join(commitErrors...)
}

// Rollback reverses every completed rename in reverse order.
func (transaction *Transaction) Rollback() error {
	if transaction == nil {
		return fmt.Errorf("plugin transaction is nil")
	}
	if transaction.committed {
		return fmt.Errorf("transaction already completed")
	}
	transaction.committed = true
	return rollbackEntries(transaction.entries)
}

func rollbackEntries(entries []renameEntry) error {
	var rollbackErrors []error
	for index := len(entries) - 1; index >= 0; index-- {
		entry := entries[index]
		if err := os.Rename(entry.to, entry.from); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback plugin %s %s -> %s: %w", entry.kind, entry.to, entry.from, err))
		}
	}
	return errors.Join(rollbackErrors...)
}

func removeConfigPlugin(cfg *config.Config, name string) {
	for index, item := range cfg.Plugins.Items {
		if item.Name == name {
			cfg.Plugins.Items = append(cfg.Plugins.Items[:index], cfg.Plugins.Items[index+1:]...)
			return
		}
	}
}

func validateManagedStaging(staging string) error {
	root := Dir()
	if root == "" {
		return fmt.Errorf("cannot determine plugins directory")
	}
	staging = filepath.Clean(staging)
	relative, err := filepath.Rel(filepath.Clean(root), staging)
	if err != nil {
		return fmt.Errorf("resolve staging path: %w", err)
	}
	if filepath.Dir(relative) != "." || !strings.HasPrefix(filepath.Base(relative), ".staging-") {
		return fmt.Errorf("staging path must be a direct .staging-* child of %s", root)
	}
	info, err := os.Lstat(staging)
	if err != nil {
		return fmt.Errorf("staging directory unavailable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("staging path must be a real directory")
	}
	return nil
}

func validateManagedFinal(name, final string) error {
	root := Dir()
	if root == "" {
		return fmt.Errorf("cannot determine plugins directory")
	}
	expected := filepath.Clean(filepath.Join(root, name))
	if filepath.Clean(final) != expected {
		return fmt.Errorf("final path must be %s", expected)
	}
	return nil
}

func validateManagedPluginSource(name, source string) error {
	if err := validateManagedFinal(name, source); err != nil {
		return err
	}
	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("plugin source unavailable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("plugin source must be a real directory")
	}
	return nil
}
