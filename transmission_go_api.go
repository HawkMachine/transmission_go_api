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

	TR_STATUS_PAUSED     = 0
	TR_STATUS_CHECK_WAIT = 1 << 0
	TR_STATUS_CHECK      = 1 << 1
	TR_STATUS_DOWNLOAD   = 1 << 2
	TR_STATUS_SEEK       = 1 << 3
	TR_STATUS_STOPPED    = 1 << 4
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
	BytesCompleted int64  `json:"bytesCompleted,omitempty"`
	Length         int64  `json:"length,omitempty"`
}

type FileStats struct {
	BytesCompleted int64 `json:"bytesCompleted,omitempty"`
	Wanted         bool  `json:"wanted,omitempty"`
	Priority       int64 `json:"priority,omitempty"`
}

type Peer struct {
	Address string `json:"address,omitempty"`
}

type Torrent struct {
	ActivityDate            int64        `json:"activityDate,omitempty"`
	AddedDate               int64        `json:"addedDate,omitempty"`
	BandwidthPriority       int64        `json:"bandwidthPriority,omitempty"`
	Comment                 string       `json:"comment,omitempty"`
	CorruptEver             int64        `json:"corruptEver,omitempty"`
	Creator                 string       `json:"creator,omitempty"`
	DateCreated             int64        `json:"dateCreated,omitempty"`
	DesiredAvailable        int64        `json:"desiredAvailable,omitempty"`
	DoneDate                int64        `json:"doneDate,omitempty"`
	DownloadDir             string       `json:"downloadDir,omitempty"`
	DownloadedEver          int64        `json:"downloadedEver,omitempty"`
	DownloadLimit           int64        `json:"downloadLimit,omitempty"`
	DownloadLimited         bool         `json:"downloadLimited,omitempty"`
	Error                   int64        `json:"error,omitempty"`
	ErrorString             string       `json:"errorString,omitempty"`
	Eta                     int64        `json:"eta,omitempty"`
	EtaIdle                 int64        `json:"etaIdle,omitempty"`
	Files                   []*File      `json:"files,omitempty"`
	FileStats               []*FileStats `json:"fileStats,omitempty"`
	HashString              string       `json:"hashString,omitempty"`
	HaveUnchecked           int64        `json:"haveUnchecked,omitempty"`
	HaveValid               int64        `json:"haveValid,omitempty"`
	HonorsSessionLimits     bool         `json:"honorsSessionLimits,omitempty"`
	Id                      int64        `json:"id,omitempty"`
	IsFinished              bool         `json:"isFinished,omitempty"`
	IsPrivate               bool         `json:"isPrivate,omitempty"`
	IsStalled               bool         `json:"isStalled,omitempty"`
	LeftUntilDone           int64        `json:"leftUntilDone,omitempty"`
	MagnetLink              string       `json:"magnetLink,omitempty"`
	ManualAnnounceTime      int64        `json:"manualAnnounceTime,omitempty"`
	MaxConnectedPeers       int64        `json:"maxConnectedPeers,omitempty"`
	MetadataPercentComplete float64      `json:"metadataPercentComplete,omitempty"`
	Name                    string       `json:"name,omitempty"`
	PeerLimit               int64        `json:"peerLimit,omitempty"`
	Peers                   []int64      `json:"peers,omitempty"`
	PeersConnected          int64        `json:"peersConnected,omitempty"`
	PeersFrom               int64        `json:"peersFrom,omitempty"`
	PeersGettingFromUs      int64        `json:"peersGettingFromUs,omitempty"`
	PeersSendingToUs        int64        `json:"peersSendingToUs,omitempty"`
	PercentDone             float64      `json:"percentDone,omitempty"`
	Pieces                  string       `json:"pieces,omitempty"`
	PieceCount              int64        `json:"pieceCount,omitempty"`
	PieceSize               int64        `json:"pieceSize,omitempty"`
	Priorities              []int64      `json:"priorities,omitempty"`
	QueuePosition           int64        `json:"queuePosition,omitempty"`
	RateDownload            int64        `json:"rateDownload,omitempty"` // B/s
	RateUpload              int64        `json:"rateUpload,omitempty"`   // B/s
	RecheckProgress         float64      `json:"recheckProgress,omitempty"`
	SecondsDownloading      int64        `json:"secondsDownloading,omitempty"`
	SecondsSeeding          int64        `json:"secondsSeeding,omitempty"`
	SeedIdleLimit           int64        `json:"seedIdleLimit,omitempty"`
	SeedIdleMode            int64        `json:"seedIdleMode,omitempty"`
	SeedRatioLimit          float64      `json:"seedRatioLimit,omitempty"`
	SeedRatioMode           int64        `json:"seedRatioMode,omitempty"`
	SizeWhenDone            int64        `json:"sizeWhenDone,omitempty"`
	StartDate               int64        `json:"startDate,omitempty"`
	Status                  int64        `json:"status,omitempty"`
	Trackers                int64        `json:"trackers,omitempty"`
	TrackerStats            int64        `json:"trackerStats,omitempty"`
	TotalSize               int64        `json:"totalSize,omitempty"`
	TorrentFile             string       `json:"torrentFile,omitempty"`
	UploadedEver            int64        `json:"uploadedEver,omitempty"`
	UploadLimit             int64        `json:"uploadLimit,omitempty"`
	UploadLimited           bool         `json:"uploadLimited,omitempty"`
	UploadRatio             float64      `json:"uploadRatio,omitempty"`
	Wanted                  int64        `json:"wanted,omitempty"`
	Webseeds                int64        `json:"webseeds,omitempty"`
	WebseedsSendingToUs     int64        `json:"webseedsSendingToUs,omitempty"`
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

// 3.0 Methods with ids with no result
type torrentRequestsRequestPayload struct {
	Ids []int64 `json:"ids,omitempty"` // Limiting the request only to numeric ids.
}

type torrentRequestsRequest struct {
	*requestBase
	Arguments *torrentRequestsRequestPayload `json:"arguments"`
}

type torrentRequestsResponse struct {
	*responseBase
}

func (t *Transmission) torrentRequests(method string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	req := torrentRequestsRequest{
		requestBase: &requestBase{
			Method: method,
			Tag:    1,
		},
		Arguments: &torrentRequestsRequestPayload{
			Ids: ids,
		},
	}
	resp := &torrentRequestsResponse{}
	err := t.doRPC(req, resp)
	if err != nil {
		return err
	}
	if resp.Result != "success" {
		return fmt.Errorf(resp.Result)
	}
	return nil
}

func torrentsToIds(torrents []*Torrent) []int64 {
	var ids []int64
	for _, t := range torrents {
		ids = append(ids, t.Id)
	}
	return ids
}

// 3.1 Start Start-Now Stop Verify Reannounce Torrent

func (t *Transmission) StartTorrents(torrents []*Torrent) error {
	return t.Start(torrentsToIds(torrents))
}

func (t *Transmission) Start(ids []int64) error {
	return t.torrentRequests("torrent-start", ids)
}

func (t *Transmission) StartNowTorrents(torrents []*Torrent) error {
	return t.StartNow(torrentsToIds(torrents))
}

func (t *Transmission) StartNow(ids []int64) error {
	return t.torrentRequests("torrent-start-now", ids)
}

func (t *Transmission) StopTorrents(torrents []*Torrent) error {
	return t.Stop(torrentsToIds(torrents))
}

func (t *Transmission) Stop(ids []int64) error {
	return t.torrentRequests("torrent-stop", ids)
}

func (t *Transmission) VerifyTorrents(torrents []*Torrent) error {
	return t.Verify(torrentsToIds(torrents))
}

func (t *Transmission) Verify(ids []int64) error {
	return t.torrentRequests("torrent-verify", ids)
}

func (t *Transmission) ReannounceTorrents(torrents []*Torrent) error {
	return t.Reannounce(torrentsToIds(torrents))
}

func (t *Transmission) Reannounce(ids []int64) error {
	return t.torrentRequests("torrent-reannounce", ids)
}

func (t *Transmission) RemoveTorrents(torrents []*Torrent) error {
	return t.Remove(torrentsToIds(torrents))
}

func (t *Transmission) Remove(ids []int64) error {
	// delete-local-content = false (default)
	return t.torrentRequests("torrent-remove", ids)
}
