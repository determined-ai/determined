type ArchiveWriter interface {
	WriteFileHeader(...) (*io.Writer, error)
	// XXX need WriteDirHeader
}

type TarArchiveWriter struct {
	tw *tar.Writer
}

func (aw TarArchiveWriter) WriteFileHeader(...) (*io.Writer, error) {
	err := aw.tw.WriteFileHeader(...)
	if err != nil {
		return nil, error
	}
	return aw.tw, nil
}

type ZipArchiveWriter struct {
	zw *zip.Writer
}

func (aw ZipArchiveWriter) WriteFileHeader(...) (*io.Writer, error) {
	return aw.zw.Create(...)
}


// S3 APIs generally require a io.WriterAt, but we can only provide an io.Writer.  We could either
// configure an elaborate buffer system to download in parallel but respond to the user serially, or
// we can configure S3 with concurrency=1, and then it promises to download sequentially [1].  Then
// we can just discard the extra arg of the WriteAt call.
//
// [1] https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#Downloader
type S3SequentialWriterAt {
	base *io.Writer
}

func (w S3SequentialWriterAt) WriteAt(p []byte, off int64) (int, error) {
	return w.base.Write(p)
}

// TGZBatchDownloadIterator implements s3's BatchDownloadIterator API.
/*	type BatchDownloadIterator interface {
		// XXX: rb is not sure if Next is called first, or if DownloadObject is first!
		Next() bool
		// XXX: rb is not sure when Err is called!
		Err() error
		DownloadObject() BatchDownloadObject
	} */
type TGZBatchDownloadIterator {
	// the objects we are writing
	objects []s3.GetObjectInput
	// the output we are writing to
	aw ArchiveWriter
	// internal state
	err error
	pos int64
	// output of ArchiveWriter for this iteration
	nextWriter *io.Writer
}

func (i *TGZBatchDownloadIterator) Next() bool {
	if i.pos > len(i.objects) {
		return false
	}
	i.pos++
	// write the header of the tar file
	// XXX: detect if blob name ends in "/", and call aw.WriteDirHeader instead
	// XXX: if blob represents a dir, continue incrementing i.pos and calling aw.WriteDirHeader
	//      until we are pointed at a non-dir
	i.nextWriter, err := i.aw.WriteFileHeader(...)
	if err != nil {
		t.err = err
		// XXX: no idea if this is how to handle an error in this iterator interface
		return false
	}
	return true
}

func (i *TGZBatchDownloadIterator) Err() error {
	return i.err
}

func (i *TGZBatchDownloadIterator) DownloadObject() s3.BatchDownloadObject {
	return s3.BatchDownloadObject {
		// Write the current object into the tarfile
		// XXX if Next() is called first, this is off-by-one, right?
		Object: &i.objects[i.pos],
		// have s3 write the file contents into the tar file
		Writer: S3SequentialWriterAt(i.nextWriter),
	}
}

func s3DownloadCheckpoint(
	c context.Context, aw: ArchiveWriter, id uuid.UUID, s3config expconf.S3Config
) error {
	// XXX why Must? prolly need to study s3 api better
	mySession := session.Must(session.NewSession())

	// XXX how to inject credentials from s3config?
	svc := s3.New(mySession)

	// trim trailing "/" on prefix
	prefix = strings.TrimRight(s3Config.Prefix(), "/")
	// add uuid to prefix
	prefix = strings.Join([]string{prefix, id.String()}, "/")
	input := ListObjectsV2Input{
		Bucket: s3Config.Bucket(),
		Prefix: prefix,
	}

	configDownload := func(d *s3manager.Downloader) {
		d.Concurrency = 1
	}

	downloader := s3manager.NewDownloader(sess, configDownload)

	var outerErr error

	readPage := func(output *ListObjectsV2Output, lastPage bool) bool {
		// Create the list of GetObjectInputs we need to implement the BatchDownloadIterator.
		objects := make([]s3.GetObjectInput, 0, len(output.Contents))
		for i, obj := range output.Contents {
			objects[i] = s3.GetObjectInput{
				bucket: s3Config.Bucket()
				key: *obj.Key  // XXX pointer deref
			}
		}

		bdi := TGZBatchDownloadIterator{
			objects: objects,
			aw: aw,
		}

		// download every bucket in this page
		outerErr = downloader.DownloadWithIterator(c, &bdi)

		// return False to stop paging
		return outerErr == nil
	}

	err :=  svc.ListObjectsV2PagesWithContext(c, &input, readPage)
	if err != nil {
		return err
	}

	return outerErr
}


// DelayedRespondOK will wait until some number of bytes are written to the stream (indicating that
// we have credentials to make the download happen) before sending a 200 OK response over the wire.
struct DelayedRespondOK {
	N int64
	c *echo.Context
	mimeType string
	responded bool
	buf []byte
}

func (d *DelayedRespondOk) Write(p []byte) (int, error) {
	if !d.responded {
		if len(d.buf) + len(p) > d.N {
			// we have enough bytes to be confident we are ok
			err := d.Finish()
			if err != nil {
				return 0, err
			}
		} else {
			// just store the buf for now
			d.buf = append(d.buf, p...)
			return len(p), nil
		}
	}
	return d.c.Response().Write(p)
}

func (d *DelayedRespondOk) Finish() error {
	if d.responded {
		return nil
	}

	// XXX figure out the mime type
	c.Response().Header().Set(echo.HeaderContentType, d.mimeType)
	c.Response().WriteHeader(http.StatusOK)
	d.responded = true

	// write the stored buf to the underlying writer
	int written = 0
	for written < len(d.buf) {
		n, err := d.c.Response().Write(d.buf[written:])
		if err != nil {
			return err
		}
		written += n
	}

	return nil
}


func (m *Master) getCheckpoint(c echo.Context, mimeType string) error {
	args := struct {
		CheckpointUUID uuid.UUID `path:"checkpoint_uuid"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return err
	}

	// find experiment id
	// XXX should we bunify this?
	expID, err := m.db.ExperimentIDForCheckpoint(args.CheckpointUUID)
	if err != nil {
		// XXX: long-term, we intend to support checkpoints which are imported, and therefore have
		//      no expID... for now we don't need to worry about it, but eventually we will want to
		//      figure out a user interface that is compatible with that long-term vision.
		// XXX: also, in the less long-term, we want to support checkpoints from non-experiment
		//      tasks, but this is less of an issue because when we support that, we will also
		//      support having checkpoint storage inside those command/notebook/whatever configs.
		return err
	}

	// find config for experiment id
	// XXX: there are two reasonable strategies we could take to find the checkpoint storage
	//      credentials.  1) we look at the credentials from the experiment where the checkpoint was
	//      saved (that's what's written here).  2) we use credentials from the determined master
	//      config.  Some users will rotate s3 keys and whatnot over time, so the master.yaml
	//      creds might be more up to date.  Other users might override the master.yaml creds in a
	//      particular experiment, for a particular storage backend, and the master.yaml creds would
	//      be irrelevant.
	//      (rb): eventually I think this needs to be a parameter that the user specifies.  We might
	//      choose one straetgy over the other for the initial prototype, whatever saas needs.
	legacyConfig, err := m.db.expID(expID)
	if err != nil {
		return err
	}

	// only send 200 OK after we've read enough from checkpoint storage to think we have permissions
	// XXX: alternative would be a synchronous check for read access to storage backend
	dw := DelayedRespondOK{
		N: 1000,
		c: c,
		mimeType: mimeType
	}

	// handle either zip or tar archives
	var aw ArchiveWriter
	switch mimeType {
	case "application/gzip":
		// build a gzip writer around the delayed writer
		gz := gzip.NewWriter(dw)

		// build a tar writer around the response writer
		tw := tar.NewWriter(gz)

		aw = TarArchiveWriter{tw}

	case "application/zip":
		// build a zip writer around the delayed writer
		zw := zip.NewWriter(dw)

		aw := ZipArchiveWriter{zw}

	default:
		panic("bug in master code: format must be tgz and zip")
	}

	ckptStorage = legacyConfig.CheckpointStorage()
	switch storage := ckptStorage.GetUnionMember().(type) {
	case expconf.S3Config:
		err := s3DownloadCheckpoint(c, aw, args.CheckpointUUID, storage)
	default:
		// XXX: handle unsupported CheckpointStorage configs in a user-friendlier way
		panic("fixme")
	}

	// just in case the whole download wasn't big enough to trigger the DelayedRespondOK, we
	// manually trigger it now
	if err != nil {
		err := dw.Finish()
	}

	// handle error from the storage-specific download function
	if err != nil {
		if dw.responded {
			// no recovering from this nicely, it'll just be a broken connection for the client
			return err
		} else {
			// XXX return a nice error code to the end user, which is still possible
			// maybe `4xx master unable to access checkpoint storage` or something
		}
	}

    return nil
}


// @Summary Get a tarball of checkpoint contents.
// @Tags Checkpoints
// @ID get-checkpoint-tgz
// @Accept  json
// @Produce  application/gzip; charset=utf-8
// @Param   checkpoint_uuid path string  true  "Checkpoint UUID"
// @Success 200 {} string ""
//nolint:godot
// @Router /checkpoints/{checkpoint_uuid}/tgz [get]
func (m *Master) getCheckpointTgz(c echo.Context) error {
	return m.getCheckpoint(c, "application/gzip")
}

// @Summary Get a zip of checkpoint contents.
// @Tags Checkpoints
// @ID get-checkpoint-zip
// @Accept  json
// @Produce  application/zip; charset=utf-8
// @Param   checkpoint_uuid path string  true  "Checkpoint UUID"
// @Success 200 {} string ""
//nolint:godot
// @Router /checkpoints/{checkpoint_uuid}/zip [get]
func (m *Master) getCheckpointZip(c echo.Context) error {
	return m.getCheckpoint(c, "application/zip")
}
