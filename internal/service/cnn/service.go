package cnn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gustian305/backend/internal/dto"
)

type Service struct {
	baseURL string
	client  *http.Client
}

func NewService(
	baseURL string,
) *Service {

	return &Service{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type PredictionCandidate struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

type PredictionResponse struct {
	Label      string                `json:"label"`
	Confidence float64               `json:"confidence"`
	Candidates []PredictionCandidate `json:"candidates"`
}

func (s *Service) PredictFile(ctx context.Context, imagePath string) (*PredictionResponse, error) {

	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	contentType, err := detectImageContentType(file)
	if err != nil {
		return nil, err
	}

	part, err := writer.CreatePart(imageFormFileHeader(filepath.Base(imagePath), contentType))
	if err != nil {
		return nil, err
	}

	if _, err = io.Copy(part, file); err != nil {
		return nil, err
	}

	if err = writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/predict",
		&body,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		raw, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf(
			"cnn service error: %s",
			string(raw),
		)
	}

	var result PredictionResponse

	if err := json.NewDecoder(
		resp.Body,
	).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *Service) ClassifyFile(ctx context.Context, imagePath string) ([]dto.DetectionCandidateRequest, error) {
	result, err := s.PredictFile(ctx, imagePath)
	if err != nil {
		return nil, err
	}

	candidates := result.Candidates
	if len(candidates) == 0 && result.Label != "" {
		candidates = []PredictionCandidate{
			{
				Label:      result.Label,
				Confidence: result.Confidence,
			},
		}
	}

	detections := make([]dto.DetectionCandidateRequest, 0, len(candidates))
	for index, candidate := range candidates {
		label := strings.TrimSpace(candidate.Label)
		if label == "" {
			continue
		}

		rank := index + 1
		detections = append(detections, dto.DetectionCandidateRequest{
			Label:      label,
			Confidence: normalizeConfidence(candidate.Confidence),
			Model:      "mobilenetv3_small",
			Source:     "server_api",
			Rank:       &rank,
		})
	}

	return detections, nil
}

func normalizeConfidence(confidence float64) float64 {
	if confidence > 1 {
		confidence = confidence / 100
	}

	if confidence < 0 {
		return 0
	}

	if confidence > 1 {
		return 1
	}

	return confidence
}

func detectImageContentType(file *os.File) (string, error) {
	buffer := make([]byte, 512)

	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return http.DetectContentType(buffer[:n]), nil
}

func imageFormFileHeader(filename string, contentType string) textproto.MIMEHeader {
	header := make(textproto.MIMEHeader)
	header.Set(
		"Content-Disposition",
		mime.FormatMediaType(
			"form-data",
			map[string]string{
				"name":     "file",
				"filename": filename,
			},
		),
	)
	header.Set("Content-Type", contentType)

	return header
}
