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
	Name           string `json:"name,omitempty"`
	BytesCompleted int    `json:"bytesCompleted,omitempty"`
	Length         int    `json:"length,omitempty"`
}

type FileStats struct {
	BytesCompleted int  `json:"bytesCompleted,omitempty"`
	Wanted         bool `json:"wanted,omitempty"`
	Priority       int  `json:"priority,omitempty"`
}

type Peer struct {
	Address string `json:"address,omitempty"`
}

type Torrent struct {
	ActivityDate            int          `json:"activityDate,omitempty"`
	AddedDate               int          `json:"addedDate,omitempty"`
	BandwidthPriority       int          `json:"bandwidthPriority,omitempty"`
	Comment                 string       `json:"comment,omitempty"`
	CorruptEver             int          `json:"corruptEver,omitempty"`
	Creator                 string       `json:"creator,omitempty"`
	DateCreated             int          `json:"dateCreated,omitempty"`
	DesiredAvailable        int          `json:"desiredAvailable,omitempty"`
	DoneDate                int          `json:"doneDate,omitempty"`
	DownloadDir             string       `json:"downloadDir,omitempty"`
	DownloadedEver          int          `json:"downloadedEver,omitempty"`
	DownloadLimit           int          `json:"downloadLimit,omitempty"`
	DownloadLimited         bool         `json:"downloadLimited,omitempty"`
	Error                   int          `json:"error,omitempty"`
	ErrorString             string       `json:"errorString,omitempty"`
	Eta                     int          `json:"eta,omitempty"`
	EtaIdle                 int          `json:"etaIdle,omitempty"`
	Files                   []*File      `json:"files,omitempty"`
	FileStats               []*FileStats `json:"fileStats,omitempty"`
	HashString              string       `json:"hashString,omitempty"`
	HaveUnchecked           int          `json:"haveUnchecked,omitempty"`
	HaveValid               int          `json:"haveValid,omitempty"`
	HonorsSessionLimits     bool         `json:"honorsSessionLimits,omitempty"`
	Id                      int          `json:"id,omitempty"`
	IsFinished              bool         `json:"isFinished,omitempty"`
	IsPrivate               bool         `json:"isPrivate,omitempty"`
	IsStalled               bool         `json:"isStalled,omitempty"`
	LeftUntilDone           int          `json:"leftUntilDone,omitempty"`
	MagnetLink              string       `json:"magnetLink,omitempty"`
	ManualAnnounceTime      int          `json:"manualAnnounceTime,omitempty"`
	MaxConnectedPeers       int          `json:"maxConnectedPeers,omitempty"`
	MetadataPercentComplete float64      `json:"metadataPercentComplete,omitempty"`
	Name                    string       `json:"name,omitempty"`
	PeerLimit               int          `json:"peerLimit,omitempty"`
	Peers                   []int        `json:"peers,omitempty"`
	PeersConnected          int          `json:"peersConnected,omitempty"`
	PeersFrom               int          `json:"peersFrom,omitempty"`
	PeersGettingFromUs      int          `json:"peersGettingFromUs,omitempty"`
	PeersSendingToUs        int          `json:"peersSendingToUs,omitempty"`
	PercentDone             float64      `json:"percentDone,omitempty"`
	Pieces                  string       `json:"pieces,omitempty"`
	PieceCount              int          `json:"pieceCount,omitempty"`
	PieceSize               int          `json:"pieceSize,omitempty"`
	Priorities              []int        `json:"priorities,omitempty"`
	QueuePosition           int          `json:"queuePosition,omitempty"`
	RateDownload            int          `json:"rateDownload,omitempty"` // B/s
	RateUpload              int          `json:"rateUpload,omitempty"`   // B/s
	RecheckProgress         float64      `json:"recheckProgress,omitempty"`
	SecondsDownloading      int          `json:"secondsDownloading,omitempty"`
	SecondsSeeding          int          `json:"secondsSeeding,omitempty"`
	SeedIdleLimit           int          `json:"seedIdleLimit,omitempty"`
	SeedIdleMode            int          `json:"seedIdleMode,omitempty"`
	SeedRatioLimit          float64      `json:"seedRatioLimit,omitempty"`
	SeedRatioMode           int          `json:"seedRatioMode,omitempty"`
	SizeWhenDone            int          `json:"sizeWhenDone,omitempty"`
	StartDate               int          `json:"startDate,omitempty"`
	Status                  int          `json:"status,omitempty"`
	Trackers                int          `json:"trackers,omitempty"`
	TrackerStats            int          `json:"trackerStats,omitempty"`
	TotalSize               int          `json:"totalSize,omitempty"`
	TorrentFile             string       `json:"torrentFile,omitempty"`
	UploadedEver            int          `json:"uploadedEver,omitempty"`
	UploadLimit             int          `json:"uploadLimit,omitempty"`
	UploadLimited           bool         `json:"uploadLimited,omitempty"`
	UploadRatio             float64      `json:"uploadRatio,omitempty"`
	Wanted                  int          `json:"wanted,omitempty"`
	Webseeds                int          `json:"webseeds,omitempty"`
	WebseedsSendingToUs     int          `json:"webseedsSendingToUs,omitempty"`
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
			Fields: []string{
				"name",
				"id",
				"totalSize",
				"eta",
				"status",
				"percentDone",
				"activityDate",
				"addedDate",
				"bandwidthPriority",
				"comment",
				"corruptEver",
				"creator",
				"dateCreated",
				"desiredAvailable",
				"doneDate",
				"downloadDir",
				"downloadedEver",
				"downloadLimit",
				"downloadLimited",
				"error",
				"errorString",
				"eta",
				"etaIdle",
				"files",
				"fileStats",
				"hashString",
				"haveUnchecked",
				"haveValid",
				"honorsSessionLimits",
				"id",
				"isFinished",
				"isPrivate",
				"isStalled",
				"leftUntilDone",
				"magnetLink",
				"manualAnnounceTime",
				"maxConnectedPeers",
				"metadataPercentComplete",
				"name",
				"peerLimit",
				//"peers",
				//"peersConnected",
				//"peersFrom",
				//"peersGettingFromUs",
				//"peersSendingToUs",
				"percentDone",
				"pieces",
				"pieceCount",
				"pieceSize",
				//"priorities",
				//"queuePosition",
				"rateDownload",
				"rateUpload",
				"recheckProgress",
				"secondsDownloading",
				"secondsSeeding",
				"seedIdleLimit",
				"seedIdleMode",
				"seedRatioLimit",
				"seedRatioMode",
				"sizeWhenDone",
				"startDate",
				"status",
				//"trackers",
				//"trackerStats",
				"totalSize",
				"torrentFile",
				"uploadedEver",
				"uploadLimit",
				"uploadLimited",
				"uploadRatio",
				//"wanted",
				//"webseeds",
				"webseedsSendingToUs",
			},
		},
	}
	resp := &getResponse{}
	err := t.doRPC(req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Result != "success" {
		return nil, fmt.Errorf(resp.Result)
	}
	return resp.Arguments.Torrents, nil
}
