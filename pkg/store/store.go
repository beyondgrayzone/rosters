package store

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"rosters/pkg/models"
)

func IssuesPath(rostersDir string) string {
	return filepath.Join(rostersDir, models.IssuesFile)
}

func PlansPath(rostersDir string) string {
	return filepath.Join(rostersDir, models.PlansFile)
}

func AcquireLock(dataFilePath string) error {
	lock := dataFilePath + ".lock"
	start := time.Now()
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			f.Close()
			return nil
		}

		if !os.IsExist(err) {
			return err
		}

		st, err := os.Stat(lock)
		if err == nil && time.Since(st.ModTime()) > time.Duration(models.LockStaleMS)*time.Millisecond {
			_ = os.Remove(lock)
			continue
		}

		if time.Since(start) > time.Duration(models.LockTimeoutMS)*time.Millisecond {
			return fmt.Errorf("timeout acquiring lock for %s", dataFilePath)
		}

		time.Sleep(time.Duration(models.LockRetryMS) * time.Millisecond)
	}
}

func ReleaseLock(dataFilePath string) {
	_ = os.Remove(dataFilePath + ".lock")
}

func WithLock[T any](dataFilePath string, fn func() (T, error)) (T, error) {
	var result T
	err := AcquireLock(dataFilePath)
	if err != nil {
		return result, err
	}
	defer ReleaseLock(dataFilePath)
	return fn()
}

func ReadIssues(rostersDir string) ([]models.Issue, error) {
	path := IssuesPath(rostersDir)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Issue{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var issues []models.Issue
	scanner := bufio.NewScanner(file)
	seen := make(map[string]int)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var issue models.Issue
		if err := json.Unmarshal([]byte(line), &issue); err == nil {
			if idx, exists := seen[issue.ID]; exists {
				issues[idx] = issue
			} else {
				seen[issue.ID] = len(issues)
				issues = append(issues, issue)
			}
		}
	}
	return issues, nil
}

func ReadPlans(rostersDir string) ([]models.Plan, error) {
	path := PlansPath(rostersDir)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Plan{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var plans []models.Plan
	scanner := bufio.NewScanner(file)
	seen := make(map[string]int)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var plan models.Plan
		if err := json.Unmarshal([]byte(line), &plan); err == nil {
			if idx, exists := seen[plan.ID]; exists {
				plans[idx] = plan
			} else {
				seen[plan.ID] = len(plans)
				plans = append(plans, plan)
			}
		}
	}
	return plans, nil
}

func AppendIssue(rostersDir string, issue models.Issue) error {
	path := IssuesPath(rostersDir)
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	data, err := json.Marshal(issue)
	if err != nil {
		return err
	}

	tempPath := fmt.Sprintf("%s.tmp.%s", path, randHex(4))
	content := string(existing)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		content += "\n"
	}
	content += string(data) + "\n"

	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func WriteIssues(rostersDir string, issues []models.Issue) error {
	path := IssuesPath(rostersDir)
	tempPath := fmt.Sprintf("%s.tmp.%s", path, randHex(4))

	f, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, issue := range issues {
		data, err := json.Marshal(issue)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return err
		}
	}

	return os.Rename(tempPath, path)
}

func WritePlans(rostersDir string, plans []models.Plan) error {
	path := PlansPath(rostersDir)
	tempPath := fmt.Sprintf("%s.tmp.%s", path, randHex(4))

	f, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, plan := range plans {
		data, err := json.Marshal(plan)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return err
		}
	}

	return os.Rename(tempPath, path)
}
