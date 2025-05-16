package s3

import (
	"encoding/xml"
)

// S3Error represents the XML structure for S3 error responses.
type S3Error struct {
	XMLName    xml.Name `xml:"Error"`
	Code       string   `xml:"Code"`
	Message    string   `xml:"Message"`
	BucketName string   `xml:"BucketName,omitempty"`
	Resource   string   `xml:"Resource,omitempty"`
	RequestID  string   `xml:"RequestId"`
	HostID     string   `xml:"HostId"`
}

type CreateBucketConfiguration struct {
	XMLName            xml.Name `xml:"CreateBucketConfiguration"`
	LocationConstraint string   `xml:"LocationConstraint,omitempty"`
}

type Bucket struct {
	XMLName      xml.Name `xml:"Bucket"`
	Name         string   `xml:"Name"`
	CreationDate string   `xml:"CreationDate"`
}

type Owner struct {
	XMLName     xml.Name `xml:"Owner"`
	ID          string   `xml:"ID"`
	DisplayName string   `xml:"DisplayName"`
}

type ListAllMyBucketsResult struct {
	XMLName               xml.Name `xml:"ListAllMyBucketsResult"`
	Owner                 Owner    `xml:"Owner"`
	Buckets               []Bucket `xml:"Buckets>Bucket"`
	IsTruncated           bool     `xml:"IsTruncated,omitempty"`
	NextContinuationToken int      `xml:"NextContinuationToken,omitempty"`
}

// ListBucketResult is the top-level structure for S3 ListObjectsV2 response
type ListBucketResult struct {
	XMLName xml.Name `xml:"ListBucketResult"`
	Xmlns   string   `xml:"xmlns,attr"`

	Name      string `xml:"Name"`
	Prefix    string `xml:"Prefix,omitempty"`
	Delimiter string `xml:"Delimiter,omitempty"`
	MaxKeys   int    `xml:"MaxKeys"`

	IsTruncated           bool   `xml:"IsTruncated"`
	NextContinuationToken string `xml:"NextContinuationToken,omitempty"`

	Contents       []ContentsType     `xml:"Contents,omitempty"`
	CommonPrefixes []CommonPrefixType `xml:"CommonPrefixes,omitempty"`

	KeyCount int `xml:"KeyCount"`

	// Echo back request parameters
	ContinuationToken string `xml:"ContinuationToken,omitempty"`
	StartAfter        string `xml:"StartAfter,omitempty"`
}

// ContentsType represents an object in the S3 bucket
type ContentsType struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified,omitempty"` // Placeholder: "2006-01-02T15:04:05.000Z"
	ETag         string `xml:"ETag,omitempty"`         // Placeholder: e.g., "\"d41d8cd98f00b204e9800998ecf8427e\""
	Size         int64  `xml:"Size,omitempty"`         // Placeholder
	StorageClass string `xml:"StorageClass,omitempty"` // e.g., "STANDARD"
}

// CommonPrefixType represents a common prefix (simulated directory)
type CommonPrefixType struct {
	Prefix string `xml:"Prefix"`
}
