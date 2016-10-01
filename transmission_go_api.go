// Attempt at implementing a Transmission RPC API in Go.
//
// https://trac.transmissionbt.com/browser/trunk/extras/rpc-spec.txt

package transmission_go_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/golang/glog"
)

const (
	csrfSessionHeader = "X-Transmission-Session-Id"
)

type Transmission struct {
	address   string
	username  string
	password  string
	sessionId string
}

func New(address, username, password string) (*Transmission, error) {
	if !strings.HasPrefix(address, "http") {
		address = fmt.Sprintf("http://%s", address)
	}
	if !strings.HasSuffix(address, "/transmission/rpc") {
		address = fmt.Sprintf("%s/transmission/rpc", address)
	}
	log.Printf("Using %s as Transmission addres", address)
	return &Transmission{
		address:  address,
		username: username,
		password: password,
	}, nil
}

type File struct {
	Name           string `json:" name,omitempty"`
	BytesCompleted int    `json:" bytesCompleted,omitempty"`
	Length         int    `json:" length,omitempty"`
}

type FileStats struct {
	BytesCompleted int  `json:" bytesCompleted,omitempty"`
	Wanted         bool `json:" wanted,omitempty"`
	Priority       int  `json:" priority,omitempty"`
}

type Torrent struct {
	ActivityDate            int          `json:"activityDate"`
	AddedDate               int          `json:"addedDate"`
	BandwidthPriority       int          `json:"bandwidthPriority"`
	Comment                 string       `json:"comment"`
	CorruptEver             int          `json:"corruptEver"`
	Creator                 string       `json:"creator"`
	DateCreated             int          `json:"dateCreated"`
	DesiredAvailable        int          `json:"desiredAvailable"`
	DoneDate                int          `json:"doneDate"`
	DownloadDir             string       `json:"downloadDir"`
	DownloadedEver          int          `json:"downloadedEver"`
	DownloadLimit           int          `json:"downloadLimit"`
	DownloadLimited         bool         `json:"downloadLimited"`
	Error                   int          `json:"error"`
	ErrorString             string       `json:"errorString"`
	Eta                     int          `json:"eta"`
	EtaIdle                 int          `json:"etaIdle"`
	Files                   []*File      `json:"files"`
	FileStats               []*FileStats `json:"fileStats"`
	HashString              string       `json:"hashString"`
	HaveUnchecked           int          `json:"haveUnchecked"`
	HaveValid               int          `json:"haveValid"`
	HonorsSessionLimits     bool         `json:"honorsSessionLimits"`
	Id                      int          `json:"id"`
	IsFinished              bool         `json:"isFinished"`
	IsPrivate               bool         `json:"isPrivate"`
	IsStalled               bool         `json:"isStalled"`
	LeftUntilDone           int          `json:"leftUntilDone"`
	MagnetLink              string       `json:"magnetLink"`
	ManualAnnounceTime      int          `json:"manualAnnounceTime"`
	MaxConnectedPeers       int          `json:"maxConnectedPeers"`
	MetadataPercentComplete float64      `json:"metadataPercentComplete"`
	Name                    string       `json:"name"`
	PeerLimit               int          `json:"peerLimit"`
	Peers                   int          `json:"peers"`
	PeersConnected          int          `json:"peersConnected"`
	PeersFrom               int          `json:"peersFrom"`
	PeersGettingFromUs      int          `json:"peersGettingFromUs"`
	PeersSendingToUs        int          `json:"peersSendingToUs"`
	PercentDone             float64      `json:"percentDone"`
	Pieces                  string       `json:"pieces"`
	PieceCount              int          `json:"pieceCount"`
	PieceSize               int          `json:"pieceSize"`
	Priorities              []int        `json:"priorities"`
	QueuePosition           int          `json:"queuePosition"`
	RateDownload            int          `json:"rateDownload"` // B/s
	RateUpload              int          `json:"rateUpload"`   // B/s
	RecheckProgress         float64      `json:"recheckProgress"`
	SecondsDownloading      int          `json:"secondsDownloading"`
	SecondsSeeding          int          `json:"secondsSeeding"`
	SeedIdleLimit           int          `json:"seedIdleLimit"`
	SeedIdleMode            int          `json:"seedIdleMode"`
	SeedRatioLimit          float64      `json:"seedRatioLimit"`
	SeedRatioMode           int          `json:"seedRatioMode"`
	SizeWhenDone            int          `json:"sizeWhenDone"`
	StartDate               int          `json:"startDate"`
	Status                  int          `json:"status"`
	Trackers                int          `json:"trackers"`
	TrackerStats            int          `json:"trackerStats"`
	TotalSize               int          `json:"totalSize"`
	TorrentFile             string       `json:"torrentFile"`
	UploadedEver            int          `json:"uploadedEver"`
	UploadLimit             int          `json:"uploadLimit"`
	UploadLimited           bool         `json:"uploadLimited"`
	UploadRatio             float64      `json:"uploadRatio"`
	Wanted                  int          `json:"wanted"`
	Webseeds                int          `json:"webseeds"`
	WebseedsSendingToUs     int          `json:"webseedsSendingToUs"`
}

type requestBase struct {
	Method string `json:"method,omitempty"`
	Tag    int    `json:"tag,omitempty"`
}

type responseBase struct {
	Result string `json:"result,omitempty"`
	Tag    int    `json:"tag,omitempty"`
}

// doRPC implements the logic for talking to the Transmission and retrying on
// 409 that contains the new session Id.

func (t *Transmission) postRequest(req interface{}) (*http.Response, error) {
	bts, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	glog.V(3).Infof("TRANSMISSION POST REQUEST  : %v\n", string(bts))

	cli := &http.Client{}
	httpReq, err := http.NewRequest("POST", t.address, bytes.NewBuffer(bts))
	httpReq.Header[csrfSessionHeader] = []string{t.sessionId}
	if err != nil {
		return nil, err
	}
	if t.username != "" && t.password != "" {
		httpReq.SetBasicAuth(t.username, t.password)
	}

	httpResp, err := cli.Do(httpReq)
	glog.V(3).Infof("TRANSMISSION POST RESPONSE : %v\n", httpResp)
	glog.V(3).Infof("TRANSMISSION POST ERROR    : %v\n", err)

	return httpResp, err
}

func (t *Transmission) doRPC(req interface{}, resp interface{}) error {
	var httpResp *http.Response
	var err error

	// If first reply fails with 409, update the session id and try again.
	httpResp, err = t.postRequest(req)
	if err != nil {
		return err
	}
	log.Printf("HTTP RESPO %v", httpResp)
	if httpResp.StatusCode == 409 {
		sessionId, ok := httpResp.Header[csrfSessionHeader]
		if !ok {
			return fmt.Errorf("409 response without %s", csrfSessionHeader)
		}
		if len(sessionId) != 1 {
			return fmt.Errorf("409 with %s, but value is empty", csrfSessionHeader)
		}
		t.sessionId = sessionId[0]
		httpResp, err = t.postRequest(req)
		if err != nil {
			return err
		}
	}

	bts, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	glog.V(2).Infof("TRANMISSION JSON RESPONSE : %v\n", string(bts))

	dec := json.NewDecoder(bytes.NewBuffer(bts))
	err = dec.Decode(resp)
	return err
}

// 3.3.  Torrent Accessors
type getRequestPayload struct {
	Ids    []int    `json:"ids,omitempty"` // Limiting the request only to numeric ids.
	Fields []string `json:"fields,omitempty"`
}

type getResponsePayload struct {
	Torrents []*Torrent `json:"torrents"`
}

type getRequest struct {
	*requestBase
	Arguments *getRequestPayload `json:"arguments"`
}

type getResponse struct {
	*responseBase
	Arguments *getResponsePayload `json:"arguments"`
}

func (t *Transmission) ListAll() ([]*Torrent, error) {
	req := getRequest{
		requestBase: &requestBase{
			Method: "torrent-get",
			Tag:    1,
		},
		Arguments: &getRequestPayload{
			Fields: []string{"name", "id", "totalSize", "eta", "status"},
		},
	}
	resp := getResponse{}
	err := t.doRPC(req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Result != "success" {
		return nil, fmt.Errorf(resp.Result)
	}
	return resp.Arguments.Torrents, nil
}
