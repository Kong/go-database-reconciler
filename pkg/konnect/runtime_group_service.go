package konnect

import (
	"context"
	"encoding/json"
)

type RuntimeGroupService service

// controlPlaneListOpt mirrors the query parameters accepted by
// /v2/control-planes: bracketed page[size]/page[number] pagination and a
// JSON:API-style filter[name][eq] filter. This differs from the legacy
// /api/* endpoints (flat size/page, no filter support), which is why it
// isn't folded into the shared ListOpt.
type controlPlaneListOpt struct {
	Size   int    `url:"page[size],omitempty"`
	Number int    `url:"page[number],omitempty"`
	Name   string `url:"filter[name][eq],omitempty"`
}

// List fetches a list of control planes from /v2/control-planes.
func (s *RuntimeGroupService) List(ctx context.Context,
	opt *ListOpt,
) ([]*RuntimeGroup, *ListOpt, error) {
	qsOpt := &controlPlaneListOpt{}
	if opt != nil {
		qsOpt.Size = opt.Size
		qsOpt.Number = opt.Page
	}

	data, next, err := s.listControlPlanes(ctx, qsOpt)
	if err != nil {
		return nil, nil, err
	}

	runtimeGroups, err := unmarshalRuntimeGroups(data)
	if err != nil {
		return nil, nil, err
	}

	return runtimeGroups, next, nil
}

// GetByName fetches the control plane with the given exact name. It returns
// nil, nil if no control plane with that name exists.
func (s *RuntimeGroupService) GetByName(ctx context.Context, name string) (*RuntimeGroup, error) {
	data, _, err := s.listControlPlanes(ctx, &controlPlaneListOpt{Name: name})
	if err != nil {
		return nil, err
	}

	runtimeGroups, err := unmarshalRuntimeGroups(data)
	if err != nil {
		return nil, err
	}

	for _, rg := range runtimeGroups {
		if rg.Name != nil && *rg.Name == name {
			return rg, nil
		}
	}
	return nil, nil
}

// listControlPlanes fetches a single page of /v2/control-planes. The
// response's pagination lives under meta.page.{number,size,total}, unlike
// the legacy /api/* endpoints' top-level page/pageCount.
func (s *RuntimeGroupService) listControlPlanes(ctx context.Context,
	opt *controlPlaneListOpt,
) ([]json.RawMessage, *ListOpt, error) {
	// page[size] is capped at 100 by the API.
	if opt.Size <= 0 || opt.Size > pageSize {
		opt.Size = pageSize
	}

	req, err := s.client.NewRequest("GET", "/v2/control-planes", opt, nil)
	if err != nil {
		return nil, nil, err
	}

	var list struct {
		Data []json.RawMessage `json:"data"`
		Meta struct {
			Page struct {
				Number int `json:"number"`
				Size   int `json:"size"`
				Total  int `json:"total"`
			} `json:"page"`
		} `json:"meta"`
	}

	_, err = s.client.Do(ctx, req, &list)
	if err != nil {
		return nil, nil, err
	}

	var next *ListOpt
	if len(list.Data) > 0 && list.Meta.Page.Size > 0 {
		totalPages := (list.Meta.Page.Total + list.Meta.Page.Size - 1) / list.Meta.Page.Size
		if list.Meta.Page.Number < totalPages {
			next = &ListOpt{
				Page: list.Meta.Page.Number + 1,
				Size: opt.Size,
			}
		}
	}

	return list.Data, next, nil
}

func unmarshalRuntimeGroups(data []json.RawMessage) ([]*RuntimeGroup, error) {
	var runtimeGroups []*RuntimeGroup
	for _, object := range data {
		b, err := object.MarshalJSON()
		if err != nil {
			return nil, err
		}
		var runtimeGroup RuntimeGroup
		if err := json.Unmarshal(b, &runtimeGroup); err != nil {
			return nil, err
		}
		runtimeGroups = append(runtimeGroups, &runtimeGroup)
	}
	return runtimeGroups, nil
}
