package dto

// APIErrorResponse adalah format umum response error API.
type APIErrorResponse struct {
	Error string `json:"error" example:"unauthorized"`
}

// APIMessageResponse adalah format umum response pesan sederhana.
type APIMessageResponse struct {
	Message string `json:"message" example:"success"`
}

// DeleteAllConversationsResponse adalah response ketika seluruh percakapan pengguna dihapus.
type DeleteAllConversationsResponse struct {
	Message      string `json:"message" example:"all conversations deleted"`
	DeletedCount int64  `json:"deleted_count" example:"3"`
}

// MessageListResponse adalah response daftar pesan pada satu percakapan.
type MessageListResponse struct {
	Items []MessageResponse `json:"items"`
}

// UploadImageResponse adalah response upload gambar hama padi.
type UploadImageResponse struct {
	Message    string                      `json:"message" example:"upload success"`
	URL        string                      `json:"url" example:"/uploads/1710000000_uuid.jpg"`
	CNNSource  string                      `json:"cnn_source" example:"on_device_flutter"`
	Detections []DetectionCandidateRequest `json:"detections"`
}
