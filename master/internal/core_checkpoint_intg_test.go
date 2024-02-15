//go:build integration
// +build integration

package internal

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/storage"
	"github.com/determined-ai/determined/master/internal/user"
	dets3 "github.com/determined-ai/determined/master/pkg/checkpoints/s3"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/modelv1"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	S3TestBucket        = "storage-unit-tests"
	S3TestBucketUSEast1 = "storage-unit-tests-us-east-1"
	S3TestPrefix        = "master/checkpoint-download"
)

var mockCheckpointContent = map[string]string{
	"emptyDir": "",
	"data.txt": "This is mock data.",
	// This long string must be longer than delayWriter.delayBytes
	"lib/big-data.txt": genLongString(1024 * 64),
	"lib/math.py":      "def triple(x):\n  return x * 3",
	"print.py":         `print("hello")`,
}

func genLongString(approxLength int) string {
	const block = "12345678223456783234567842345678\n"
	var sb strings.Builder

	for j := 0; j < approxLength; j += len(block) {
		sb.WriteString(block)
	}
	return sb.String()
}

func createMockCheckpointS3(bucket string, prefix string) error {
	region, err := dets3.GetS3BucketRegion(context.TODO(), bucket, nil)
	if err != nil {
		return err
	}
	sess, err := session.NewSession(&aws.Config{
		Region: &region,
	})
	if err != nil {
		return err
	}
	s3client := s3.New(sess)

	for k, v := range mockCheckpointContent {
		_, err = s3client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(prefix + "/" + k),
			Body:   aws.ReadSeekCloser(strings.NewReader(v)),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func checkTar(t *testing.T, rec *httptest.ResponseRecorder, id string) {
	require.Equal(t, strconv.Itoa(rec.Body.Len()), rec.Header().Get("Content-Length"))
	content := rec.Body
	tr := tar.NewReader(content)
	gotMap := make(map[string]string)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		require.NoError(t, err, "failed to read record header")
		buf := &strings.Builder{}
		if hdr.Size > 0 {
			_, err := io.Copy(buf, tr) //nolint: gosec
			require.NoError(t, err, "failed to read content of file", hdr.Name)
		}
		gotMap[hdr.Name] = buf.String()
	}
	require.Equal(t, mockCheckpointContent, gotMap)
}

func checkTgz(t *testing.T, rec *httptest.ResponseRecorder, id string) {
	content := rec.Body
	zr, err := gzip.NewReader(content)
	require.NoError(t, err, "failed to create a gzip reader")
	tr := tar.NewReader(zr)
	gotMap := make(map[string]string)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		require.NoError(t, err, "failed to read record header")
		buf := &strings.Builder{}
		if hdr.Size > 0 {
			_, err := io.Copy(buf, tr) //nolint: gosec
			require.NoError(t, err, "failed to read content of file", hdr.Name)
		}
		gotMap[hdr.Name] = buf.String()
	}
	require.Equal(t, mockCheckpointContent, gotMap)
}

func checkZip(t *testing.T, rec *httptest.ResponseRecorder, id string) {
	content := rec.Body.String()
	zr, err := zip.NewReader(strings.NewReader(content), int64(len(content)))
	require.NoError(t, err, "failed to create a zip reader")
	gotMap := make(map[string]string)
	for _, f := range zr.File {
		buf := &strings.Builder{}
		rc, err := f.Open()
		require.NoError(t, err, "unable to decompress file", f.Name)
		_, err = io.Copy(buf, rc) //nolint: gosec
		require.NoError(t, err, "unable to read content of file", f.Name)
		require.NoError(t, rc.Close())
		gotMap[f.Name] = buf.String()
	}
	require.Equal(t, mockCheckpointContent, gotMap)
}

func addMockCheckpointDB(t *testing.T, pgDB *db.PgDB, id uuid.UUID, bucket string) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	user := db.RequireMockUser(t, pgDB)
	// Using a different path than DefaultTestSrcPath since we are one level up than most db tests
	exp := mockExperimentS3(t, pgDB, user, "../../examples/tutorials/mnist_pytorch", bucket)
	tr, task := db.RequireMockTrial(t, pgDB, exp)
	allocation := db.RequireMockAllocation(t, pgDB, task.TaskID)
	// Create checkpoints
	checkpoint := db.MockModelCheckpoint(id, allocation)
	err := db.AddCheckpointMetadata(context.TODO(), &checkpoint, tr.ID)
	require.NoError(t, err)
}

func createCheckpoint(t *testing.T, pgDB *db.PgDB, bucket string) (string, error) {
	id := uuid.New()
	addMockCheckpointDB(t, pgDB, id, bucket)
	err := createMockCheckpointS3(bucket, S3TestPrefix+"/"+id.String())
	return id.String(), err
}

func setupCheckpointTestEcho(t *testing.T) (
	*apiServer, echo.Context, *httptest.ResponseRecorder,
) {
	api, _, _ := setupAPITest(t, nil)
	e := echo.New()
	rec := httptest.NewRecorder()
	ctx := &detContext.DetContext{Context: e.NewContext(nil, rec)}

	admin, err := user.ByUsername(context.TODO(), "admin")
	require.NoError(t, err)
	ctx.SetUser(*admin)

	return api, ctx, rec
}

func TestGetCheckpointEcho(t *testing.T) {
	testGetCheckpointEcho(t, S3TestBucket)
}

func TestGetCheckpointEchoUSEast1(t *testing.T) {
	testGetCheckpointEcho(t, S3TestBucketUSEast1)
}

func testGetCheckpointEcho(t *testing.T, bucket string) {
	gitBranch := os.Getenv("CIRCLE_BRANCH")
	if gitBranch == "" || strings.HasPrefix(gitBranch, "pull/") {
		t.Skipf("skipping test %s in a forked repo (branch: %s) due to lack of credentials",
			t.Name(), gitBranch)
	}
	cases := []struct {
		DenyFuncName string
		IDToReqCall  func() error
		Params       []any
	}{
		{"CanGetCheckpointTar", func() error {
			api, ctx, rec := setupCheckpointTestEcho(t)
			id, err := createCheckpoint(t, api.m.db, bucket)
			if err != nil {
				return err
			}
			ctx.SetParamNames("checkpoint_uuid")
			ctx.SetParamValues(id)
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/", nil))
			ctx.Request().Header.Set("Accept", MIMEApplicationXTar)
			err = api.m.getCheckpoint(ctx)
			require.NoError(t, err, "API call returns error")
			checkTar(t, rec, id)
			return err
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
		{"CanGetCheckpointTgz", func() error {
			api, ctx, rec := setupCheckpointTestEcho(t)
			id, err := createCheckpoint(t, api.m.db, bucket)
			if err != nil {
				return err
			}
			ctx.SetParamNames("checkpoint_uuid")
			ctx.SetParamValues(id)
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/", nil))
			ctx.Request().Header.Set("Accept", MIMEApplicationGZip)
			err = api.m.getCheckpoint(ctx)
			require.NoError(t, err, "API call returns error")
			checkTgz(t, rec, id)
			return err
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
		{"CanGetCheckpointZip", func() error {
			api, ctx, rec := setupCheckpointTestEcho(t)
			id, err := createCheckpoint(t, api.m.db, bucket)
			if err != nil {
				return err
			}
			ctx.SetParamNames("checkpoint_uuid")
			ctx.SetParamValues(id)
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/", nil))
			ctx.Request().Header.Set("Accept", MIMEApplicationZip)
			err = api.m.getCheckpoint(ctx)
			require.NoError(t, err, "API call returns error")
			checkZip(t, rec, id)
			return err
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
	}

	for _, curCase := range cases {
		require.NoError(t, curCase.IDToReqCall())
	}
}

// nolint: exhaustruct
func TestGetCheckpointStorageConfig(t *testing.T) {
	api, _, _ := setupCheckpointTestEcho(t)
	ctx := context.Background()

	checkpointID := uuid.New()
	addMockCheckpointDB(t, api.m.db, checkpointID, S3TestBucket)

	// Fallback to expconf when no storageID.
	conf, err := api.m.getCheckpointStorageConfig(ctx, checkpointID)
	require.NoError(t, err)
	require.Equal(t, schemas.WithDefaults(&expconf.CheckpointStorageConfig{
		RawS3Config: &expconf.S3ConfigV0{
			RawBucket: ptrs.Ptr(S3TestBucket),
			RawPrefix: ptrs.Ptr(S3TestPrefix),
		},
	}), conf)

	// Otherwise lookup storageID config.
	expected := &expconf.CheckpointStorageConfigV0{
		RawS3Config: &expconf.S3ConfigV0{
			RawBucket: ptrs.Ptr(uuid.New().String()),
		},
	}
	storageID, err := storage.AddBackend(ctx, expected)
	require.NoError(t, err)

	_, err = db.Bun().NewUpdate().Table("checkpoints_v2").
		Where("uuid = ?", checkpointID).
		Set("storage_id = ?", storageID).
		Exec(ctx)
	require.NoError(t, err)

	conf, err = api.m.getCheckpointStorageConfig(ctx, checkpointID)
	require.NoError(t, err)
	require.Equal(t, expected, conf)
}

// TestGetCheckpointEchoExpErr expects specific errors are returned for each check.
func TestGetCheckpointEchoExpErr(t *testing.T) {
	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id string) error
		Params       []any
	}{
		{"CanGetCheckpointTgz", func(id string) error {
			api, ctx, _ := setupCheckpointTestEcho(t)
			ctx.SetParamNames("checkpoint_uuid")
			ctx.SetParamValues(id)
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/", nil))
			ctx.Request().Header.Set("Accept", MIMEApplicationGZip)
			return api.m.getCheckpoint(ctx)
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
		{"CanGetCheckpointZip", func(id string) error {
			api, ctx, _ := setupCheckpointTestEcho(t)
			ctx.SetParamNames("checkpoint_uuid")
			ctx.SetParamValues(id)
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/", nil))
			ctx.Request().Header.Set("Accept", MIMEApplicationZip)
			return api.m.getCheckpoint(ctx)
		}, []any{mock.Anything, mock.Anything}},
	}

	for _, curCase := range cases {
		// Checkpoint not found
		require.Equal(t, echo.NewHTTPError(http.StatusNotFound,
			`checkpoint '7e0bad2c-b3f6-4988-916c-eb3081b19db0' not found`),
			curCase.IDToReqCall("7e0bad2c-b3f6-4988-916c-eb3081b19db0"))

		// Invalid checkpoint UUID
		require.Equal(t,
			echo.NewHTTPError(http.StatusBadRequest,
				"unable to parse checkpoint UUID badbad-b3f6-4988-916c-eb5581b19db0: "+
					"invalid UUID length: 34"),
			curCase.IDToReqCall("badbad-b3f6-4988-916c-eb5581b19db0"))
	}
}

func RegisterCheckpointAsModelVersion(ctx context.Context, t *testing.T, pgDB *db.PgDB, ckptID uuid.UUID,
) *modelv1.ModelVersion {
	require.NoError(t, etc.SetRootPath("../../master/static/srv"))
	retCkpt, err := db.GetCheckpoint(ctx, ckptID.String())
	require.NoError(t, err)
	user := db.RequireMockUser(t, pgDB)
	// Insert a model.
	now := time.Now()
	mdl := db.Model{
		Name:            uuid.NewString(),
		Description:     "some important model",
		CreationTime:    now,
		LastUpdatedTime: now,
		Labels:          []string{"some other label"},
		UserID:          user.ID,
		WorkspaceID:     1,
	}
	emptyMetadata := []byte(`{}`)
	mdlNotes := "some notes"
	pmdl, err := db.InsertModel(ctx, mdl.Name, mdl.Description, emptyMetadata,
		strings.Join(mdl.Labels, ","), mdlNotes, user.ID, mdl.WorkspaceID)
	require.NoError(t, err)

	// Register checkpoint as a model version.
	expected := &modelv1.ModelVersion{
		Model:      pmdl,
		Checkpoint: retCkpt,
		Name:       "some name",
		Comment:    "empty",
		Username:   user.Username,
		Labels:     []string{"some label"},
		Notes:      "some notes",
	}
	mv, err := db.InsertModelVersion(ctx, pmdl.Id, ckptID.String(), expected.Name, expected.Comment,
		emptyMetadata, strings.Join(expected.Labels, ","), expected.Notes, user.ID,
	)
	require.NoError(t, err)
	return mv
}

func TestAuthZCheckpointsEcho(t *testing.T) {
	api, authZExp, _, curUser, _ := setupExpAuthTest(t, nil)
	authZModel := getMockModelAuth()
	ctx := newTestEchoContext(curUser)

	checkpointUUID := uuid.New()
	checkpointID := checkpointUUID.String()

	ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/", nil))
	ctx.Request().Header.Set("Accept", MIMEApplicationZip)
	ctx.SetParamNames("checkpoint_uuid")
	ctx.SetParamValues(checkpointID)

	// Not found same as permission denied.
	require.Equal(t, apiPkg.NotFoundErrs("checkpoint", fmt.Sprint(checkpointUUID), false),
		api.m.getCheckpoint(ctx))

	addMockCheckpointDB(t, api.m.db, checkpointUUID, S3TestBucket)
	RegisterCheckpointAsModelVersion(context.TODO(), t, api.m.db, checkpointUUID)

	authZExp.On("CanGetExperiment", mock.Anything, curUser,
		mock.Anything).Return(authz2.PermissionDeniedError{}).Once()

	authZModel.On("CanGetModel", mock.Anything, mock.Anything,
		mock.Anything, mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
	require.Equal(t, apiPkg.NotFoundErrs("checkpoint", fmt.Sprint(checkpointUUID), false),
		api.m.getCheckpoint(ctx))

	// need to make the model auth fail too for actual to be the expected
	expectedErr := fmt.Errorf("canGetExperimentError")
	authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
		Return(expectedErr).Once()
	authZModel.On("CanGetModel", mock.Anything, mock.Anything,
		mock.Anything, mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
	require.Equal(t, expectedErr, api.m.getCheckpoint(ctx))

	expectedErr = echo.NewHTTPError(http.StatusForbidden, "canGetArtifactsError")
	authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).Return(nil).Once()
	authZModel.On("CanGetModel", mock.Anything, mock.Anything,
		mock.Anything, mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
	authZExp.On("CanGetExperimentArtifacts", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canGetArtifactsError")).Once()
	require.Equal(t, expectedErr, api.m.getCheckpoint(ctx))
}

// nolint: exhaustruct
func mockExperimentS3(
	t *testing.T, pgDB *db.PgDB, user model.User, folderPath, bucket string,
) *model.Experiment {
	cfg := schemas.WithDefaults(expconf.ExperimentConfigV0{
		RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
			RawS3Config: &expconf.S3ConfigV0{
				RawBucket: aws.String(bucket),
				RawPrefix: aws.String(S3TestPrefix),
			},
		},
		RawEntrypoint: &expconf.EntrypointV0{
			RawEntrypoint: ptrs.Ptr("model.Classifier"),
		},
		RawHyperparameters: map[string]expconf.HyperparameterV0{
			"global_batch_size": {
				RawConstHyperparameter: &expconf.ConstHyperparameterV0{
					RawVal: ptrs.Ptr(1),
				},
			},
		},
		RawSearcher: &expconf.SearcherConfigV0{
			RawSingleConfig: &expconf.SingleConfigV0{
				RawMaxLength: &expconf.LengthV0{
					Unit:  expconf.Batches,
					Units: 1,
				},
			},
			RawMetric: ptrs.Ptr("okness"),
		},
	})

	exp := model.Experiment{
		JobID:     model.NewJobID(),
		State:     model.ActiveState,
		Config:    cfg.AsLegacy(),
		StartTime: time.Now().Add(-time.Hour),
		OwnerID:   &user.ID,
		Username:  user.Username,
		ProjectID: 1,
	}
	err := pgDB.AddExperiment(&exp, db.ReadTestModelDefiniton(t, folderPath), cfg)
	require.NoError(t, err, "failed to add experiment")
	return &exp
}
