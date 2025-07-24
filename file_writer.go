package main

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
)

type FileWriter struct {
	app          *App
	file         *os.File
	bufferedFile *bufio.Writer
	zipWriter    *gzip.Writer
	csvWriter    *csv.Writer
	filePath     string
	baseFilePath string
	tempFilePath string
	rowCount     int
	fileIndex    int
	isFinalized  bool
}

func newFileWriter(app *App, filePath string) (*FileWriter, error) {
	fw := &FileWriter{
		app:          app,
		baseFilePath: strings.TrimSuffix(filePath, ".csv.gz"),
		fileIndex:    1,
	}

	err := fw.initFile(filePath)
	if err != nil {
		return nil, err
	}

	return fw, nil
}

func (fw *FileWriter) initFile(filePath string) error {
	fw.filePath = filePath
	fw.tempFilePath = fw.filePath + ".tmp"

	file, err := os.Create(fw.tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file %s: %v", fw.tempFilePath, err)
	}

	fw.file = file
	fw.bufferedFile = bufio.NewWriterSize(file, 1<<20)
	fw.zipWriter = gzip.NewWriter(fw.bufferedFile)
	fw.csvWriter = csv.NewWriter(fw.zipWriter)
	fw.rowCount = 0
	fw.isFinalized = false

	return nil
}

func (fw *FileWriter) writeHeaders(headers []string) error {
	if fw.csvWriter == nil {
		return fmt.Errorf("file writer not initialized")
	}
	return fw.csvWriter.Write(headers)
}

func (fw *FileWriter) writeRow(row []string, monthOfData string, filesToUpload map[string]map[string]struct{}) error {
	if fw.rowCount >= fw.app.MaxFileRows {
		err := fw.rotateFile(monthOfData, filesToUpload)
		if err != nil {
			return err
		}

		err = fw.csvWriter.Write(fw.app.getCSVHeaders())
		if err != nil {
			return fmt.Errorf("failed to write headers after rotation: %v", err)
		}
	}

	if fw.csvWriter == nil {
		return fmt.Errorf("file writer not initialized")
	}

	err := fw.csvWriter.Write(row)
	if err != nil {
		return fmt.Errorf("failed to write CSV row: %v", err)
	}

	fw.rowCount++
	return nil
}

func (fw *FileWriter) rotateFile(monthOfData string, filesToUpload map[string]map[string]struct{}) error {
	err := fw.finalizeFile(monthOfData, filesToUpload)
	if err != nil {
		return fmt.Errorf("failed to finalize file during rotation: %v", err)
	}

	fw.fileIndex++
	newFilePath := fmt.Sprintf("%s-%d.csv.gz", fw.baseFilePath, fw.fileIndex)

	err = fw.initFile(newFilePath)
	if err != nil {
		return fmt.Errorf("failed to initialize new file after rotation: %v", err)
	}

	return nil
}

func (fw *FileWriter) finalizeFile(monthOfData string, filesToUpload map[string]map[string]struct{}) error {
	err := fw.close()
	if err != nil {
		fw.cleanup()
		return fmt.Errorf("failed to close file during finalization: %v", err)
	}

	if fw.rowCount > 0 {
		err = os.Rename(fw.tempFilePath, fw.filePath)
		if err != nil {
			fw.cleanup()
			return fmt.Errorf("failed to rename temp file to final: %v", err)
		}

		fw.isFinalized = true

		if filesToUpload[monthOfData] == nil {
			filesToUpload[monthOfData] = make(map[string]struct{})
		}
		filesToUpload[monthOfData][fw.filePath] = struct{}{}
		log.Printf("Generated file: %s (%d rows)", fw.filePath, fw.rowCount)
	} else {
		fw.cleanup()
	}

	return nil
}

func (fw *FileWriter) close() error {
	var errors []error

	if fw.csvWriter != nil {
		fw.csvWriter.Flush()
		if err := fw.csvWriter.Error(); err != nil {
			errors = append(errors, fmt.Errorf("CSV writer error: %v", err))
		}
	}

	if fw.zipWriter != nil {
		if err := fw.zipWriter.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close gzip writer: %v", err))
		}
	}

	if fw.bufferedFile != nil {
		if err := fw.bufferedFile.Flush(); err != nil {
			errors = append(errors, fmt.Errorf("failed to flush buffered writer: %v", err))
		}
	}

	if fw.file != nil {
		if err := fw.file.Sync(); err != nil {
			errors = append(errors, fmt.Errorf("failed to sync file to disk: %v", err))
		}
		if err := fw.file.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close file: %v", err))
		}
	}

	fw.file = nil
	fw.bufferedFile = nil
	fw.zipWriter = nil
	fw.csvWriter = nil

	if len(errors) > 0 {
		var errorMessages []string
		for _, err := range errors {
			errorMessages = append(errorMessages, err.Error())
		}
		return fmt.Errorf("multiple errors during close: %s", strings.Join(errorMessages, "; "))
	}

	return nil
}

func (fw *FileWriter) cleanup() {
	if fw.tempFilePath != "" && !fw.isFinalized {
		if err := os.Remove(fw.tempFilePath); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to cleanup temp file %s: %v", fw.tempFilePath, err)
		}
	}
}

func validateGzipHeaders(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	_, err = gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("invalid gzip format: %v", err)
	}

	return nil
}
