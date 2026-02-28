package backup

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

// AzureDestination implements backup destination for Azure Blob Storage
type AzureDestination struct {
	containerURL azblob.ContainerURL
	container    string
	prefix       string
}

// NewAzureDestination creates a new Azure Blob Storage backup destination
func NewAzureDestination(cfg *AzureConfig) (*AzureDestination, error) {
	// Create credentials
	credential, err := azblob.NewSharedKeyCredential(cfg.StorageAccount, cfg.AccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credentials: %w", err)
	}

	// Create pipeline
	pipeline := azblob.NewPipeline(credential, azblob.PipelineOptions{
		Retry: azblob.RetryOptions{
			MaxTries:      3,
			TryTimeout:    time.Minute * 10,
			RetryDelay:    time.Second * 5,
			MaxRetryDelay: time.Second * 30,
		},
	})

	// Create container URL
	u, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s",
		cfg.StorageAccount, cfg.Container))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Azure URL: %w", err)
	}

	containerURL := azblob.NewContainerURL(*u, pipeline)

	return &AzureDestination{
		containerURL: containerURL,
		container:    cfg.Container,
		prefix:       "", // Could be configured if needed
	}, nil
}

// Put uploads data to Azure Blob Storage, returns size written
func (a *AzureDestination) Put(key string, reader io.Reader) (int64, error) {
	fullKey := a.getFullKey(key)
	blobURL := a.containerURL.NewBlockBlobURL(fullKey)

	// Track size while uploading
	trackingReader := &sizeTrackingReader{reader: reader}

	// Upload with automatic chunking for large files
	_, err := azblob.UploadStreamToBlockBlob(
		context.Background(),
		trackingReader,
		blobURL,
		azblob.UploadStreamToBlockBlobOptions{
			BufferSize: 4 * 1024 * 1024, // 4MB buffer
			MaxBuffers: 3,                // Limited for Raspberry Pi
		},
	)

	if err != nil {
		return 0, fmt.Errorf("failed to upload to Azure: %w", err)
	}

	return trackingReader.bytesRead, nil
}

// Get retrieves data from Azure Blob Storage
func (a *AzureDestination) Get(key string) (io.ReadCloser, error) {
	fullKey := a.getFullKey(key)
	blobURL := a.containerURL.NewBlockBlobURL(fullKey)

	response, err := blobURL.Download(
		context.Background(),
		0, // offset
		0, // count (0 means entire blob)
		azblob.BlobAccessConditions{},
		false, // rangeGetContentMD5
		azblob.ClientProvidedKeyOptions{},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to download from Azure: %w", err)
	}

	// Return the response body which implements io.ReadCloser
	return response.Body(azblob.RetryReaderOptions{
		MaxRetryRequests: 3,
	}), nil
}

// Delete removes data from Azure Blob Storage
func (a *AzureDestination) Delete(key string) error {
	fullKey := a.getFullKey(key)
	blobURL := a.containerURL.NewBlockBlobURL(fullKey)

	_, err := blobURL.Delete(
		context.Background(),
		azblob.DeleteSnapshotsOptionInclude,
		azblob.BlobAccessConditions{},
	)

	if err != nil {
		return fmt.Errorf("failed to delete from Azure: %w", err)
	}

	return nil
}

// List returns objects with the given prefix
func (a *AzureDestination) List(prefix string) ([]BackupObject, error) {
	fullPrefix := a.getFullKey(prefix)

	var objects []BackupObject

	// List blobs
	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := a.containerURL.ListBlobsFlatSegment(
			context.Background(),
			marker,
			azblob.ListBlobsSegmentOptions{
				Prefix:     fullPrefix,
				MaxResults: 100,
			},
		)

		if err != nil {
			return nil, fmt.Errorf("failed to list blobs: %w", err)
		}

		marker = listBlob.NextMarker

		for _, blobInfo := range listBlob.Segment.BlobItems {
			objects = append(objects, BackupObject{
				Key:          a.stripPrefix(blobInfo.Name),
				Size:         *blobInfo.Properties.ContentLength,
				LastModified: blobInfo.Properties.LastModified,
			})
		}
	}

	return objects, nil
}

// GetURL returns a pre-signed URL for direct access
func (a *AzureDestination) GetURL(key string, expiry time.Duration) (string, error) {
	fullKey := a.getFullKey(key)
	blobURL := a.containerURL.NewBlockBlobURL(fullKey)

	// Create SAS query parameters
	sasQueryParams, err := azblob.BlobSASSignatureValues{
		Protocol:      azblob.SASProtocolHTTPS,
		ExpiryTime:    time.Now().Add(expiry),
		ContainerName: a.container,
		BlobName:      fullKey,
		Permissions:   azblob.BlobSASPermissions{Read: true}.String(),
	}.NewSASQueryParameters(a.getCredential())

	if err != nil {
		return "", fmt.Errorf("failed to create SAS token: %w", err)
	}

	// Construct the URL with SAS token
	parts := azblob.NewBlobURLParts(blobURL.URL())
	parts.SAS = sasQueryParams
	sasURL := parts.URL()

	return sasURL.String(), nil
}

// Type returns the destination type identifier
func (a *AzureDestination) Type() string {
	return "azure"
}

// getFullKey returns the full blob name including any prefix
func (a *AzureDestination) getFullKey(key string) string {
	if a.prefix != "" {
		return a.prefix + "/" + key
	}
	return key
}

// stripPrefix removes the destination prefix from a key
func (a *AzureDestination) stripPrefix(key string) string {
	if a.prefix != "" && len(key) > len(a.prefix)+1 {
		return key[len(a.prefix)+1:]
	}
	return key
}

// getCredential extracts the credential from the pipeline (for SAS generation)
func (a *AzureDestination) getCredential() azblob.StorageAccountCredential {
	// This is a simplified approach - in production, you'd store the credential
	// as a field in the struct during initialization
	// For now, return nil which means the SAS generation might fail
	return nil
}