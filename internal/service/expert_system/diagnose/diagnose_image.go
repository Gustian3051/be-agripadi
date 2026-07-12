package diagnose

import (
	"errors"

	"github.com/gustian305/backend/internal/dto"
)

type ImageService struct{}

func NewImageService() *ImageService {
	return &ImageService{}
}

func (s *ImageService) SelectPrimaryCandidate(detections []dto.DetectionCandidateRequest) (*dto.DetectionCandidateRequest, error) {

	if len(detections) == 0 {

		return nil,
			errors.New(
				"no detection candidates",
			)
	}

	best := detections[0]

	for _, item := range detections {

		if item.Confidence >
			best.Confidence {

			best = item
		}
	}

	return &best, nil
}

func (s *ImageService) IsValidConfidence(confidence float64) bool {

	return confidence >= 0.5
}