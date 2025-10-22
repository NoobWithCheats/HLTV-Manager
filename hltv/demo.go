package hltv

import (
	log "HLTV-Manager/logger"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (h *HLTV) DemoControl() error {
	err := h.LoadDemosFromFolder()
	if err != nil {
		return err
	}

	sort.Slice(h.Demos, func(i, j int) bool {
		dateI, errI := time.Parse("2006.01.02 15:04", h.Demos[i].Date+" "+h.Demos[i].Time)
		if errI != nil {
			log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Ошибка парсинга для демки: %d %v", h.ID, h.Settings.Name, h.Demos[i].ID, errI)
			return false
		}

		dateJ, errJ := time.Parse("2006.01.02 15:04", h.Demos[j].Date+" "+h.Demos[j].Time)
		if errJ != nil {
			log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Ошибка парсинга для демки: %d %v", h.ID, h.Settings.Name, h.Demos[i].ID, errI)
			return false
		}

		return dateI.After(dateJ)
	})

	err = h.DeleteOldDemos()
	if err != nil {
		return err
	}

	return nil
}

func (h *HLTV) LoadDemosFromFolder() error {
	var demos []Demos

	var id int

	fmt.Println(h.Settings.DemoDir)

	err := filepath.Walk(h.Settings.DemoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Error accessing file: %v", h.ID, h.Settings.Name, err)
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".zip") {
			id++
			demo, err := parseDemoFilename(info.Name())
			if err != nil {
				log.WarningLogger.Printf("HLTV (ID: %d, Name: %s) Error parsing file: %s, %v", h.ID, h.Settings.Name, info.Name(), err)
				return nil
			}
			demo.ID = id
			demos = append(demos, demo)
		}

		return nil
	})

	if err != nil {
		log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Failed to walk through folder: %v", h.ID, h.Settings.Name, err)
		return err
	}

	h.Demos = demos
	return nil
}

func (h *HLTV) DeleteOldDemos() error {
	now := time.Now()

	for _, demo := range h.Demos {
		demoDate, err := time.Parse("2006.01.02 15:04", demo.Date+" "+demo.Time)
		if err != nil {
			log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Failed to parse date for demo %s: %v", h.ID, h.Settings.Name, demo.Name, err)
			return err
		}

		maxDemoDay, err := strconv.Atoi(h.Settings.MaxDemoDay)
		if err != nil {
			log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Error converting MaxDemoDay for demo %s: %v", h.ID, h.Settings.Name, demo.Name, err)
			return err
		}

		if now.Sub(demoDate).Hours() > float64(maxDemoDay*24) {
			demoPath := filepath.Join(h.Settings.DemoDir, demo.Name)
			err := os.Remove(demoPath)
			if err != nil {
				log.ErrorLogger.Printf("HLTV (ID: %d, Name: %s) Failed to remove old demo %s: %v", h.ID, h.Settings.Name, demo.Name, err)
				return err
			}
			// TODO: debug
			log.InfoLogger.Printf("HLTV (ID: %d, Name: %s) Removed old demo: %s", h.ID, h.Settings.Name, demo.Name)
		}
	}

	return nil
}

func (h *HLTV) GetDemoFileName(demoID int) (string, error) {
	var demo Demos
	for _, d := range h.Demos {
		if d.ID == demoID {
			demo = d
			break
		}
	}

	datePart := strings.ReplaceAll(demo.Date, ".", "")[2:]
	timePart := strings.ReplaceAll(demo.Time, ":", "")

	return fmt.Sprintf("%s-%s-%s.zip", h.Settings.DemoName, datePart+timePart, demo.Map), nil
}
