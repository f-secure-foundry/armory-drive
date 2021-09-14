// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path"

	"github.com/f-secure-foundry/armory-drive-log/api"
	"github.com/f-secure-foundry/armory-drive-log/api/verify"
	"github.com/f-secure-foundry/armory-drive/assets"

	"github.com/google/go-github/v34/github"
	"github.com/google/trillian-examples/formats/log"
	"github.com/google/trillian-examples/serverless/client"
	"github.com/google/trillian/merkle/rfc6962"
	"golang.org/x/mod/sumdb/note"
)

func verifyRelease(release *github.RepositoryRelease, a *releaseAssets) (err error) {
	var oldCP *log.Checkpoint
	var checkpoints []log.Checkpoint

	ctx := context.Background()

	if len(assets.LogPublicKey) == 0 {
		return errors.New("installer compiled without LOG_PUBKEY, could not verify release")
	}

	logSigV, err := note.NewVerifier(string(assets.LogPublicKey))

	if err != nil {
		return
	}

	// TODO(al,andrea): this should be an asset provisioned at the same time the log URL and PubKey are.
	origin := api.OriginV0
	newCP, newCPRaw, err := client.FetchCheckpoint(ctx, logFetcher, logSigV, origin)

	if err != nil {
		return
	}

	if cacheDir, err := os.UserCacheDir(); err == nil {
		p := path.Join(cacheDir, checkpointCachePath)

		buf, err := os.ReadFile(p)

		if err == nil {
			oldCP = &log.Checkpoint{}
			oldCP.Unmarshal(buf)
		}

		defer func() {
			if len(newCPRaw) > 0 {
				_ = os.WriteFile(p, newCPRaw, 0600)
			}
		}()
	}

	if oldCP != nil {
		checkpoints = append(checkpoints, *oldCP)
	}

	if len(checkpoints) > 0 {
		checkpoints = append(checkpoints, *newCP)
		err = client.CheckConsistency(ctx, rfc6962.DefaultHasher, logFetcher, checkpoints)
	}

	return verifyProof(a)
}

func verifyProof(a *releaseAssets) (err error) {
	if len(a.log) == 0 {
		return errors.New("missing proof")
	}

	pb := &api.ProofBundle{}

	if err = json.Unmarshal(a.log, pb); err != nil {
		return
	}

	logSigV, err := note.NewVerifier(string(assets.LogPublicKey))

	if err != nil {
		return
	}

	frSigV, err := note.NewVerifier(string(assets.FRPublicKey))

	if err != nil {
		return
	}

	imxHash := sha256.Sum256(a.imx)
	csfHash := sha256.Sum256(a.csf)

	hashes := map[string][]byte{
		imxPath: imxHash[:],
		csfPath: csfHash[:],
	}

	if err = verify.Bundle(*pb, api.Checkpoint{}, logSigV, frSigV, hashes); err != nil {
		return
	}

	// leaf hashes are not needed so we can save space
	pb.LeafHashes = nil

	return
}
