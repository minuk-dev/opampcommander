package ginutil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
	"github.com/minuk-dev/opampcommander/pkg/selector"
)

func parseSelectorsFor(
	t *testing.T, query string, allowed []string,
) (ginutil.Selectors, bool, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/hosts?"+query, nil)

	selectors, ok := ginutil.ParseSelectors(ctx, allowed)

	return selectors, ok, recorder
}

func TestParseSelectors_Empty(t *testing.T) {
	t.Parallel()

	selectors, ok, recorder := parseSelectorsFor(t, "", []string{"spec.platform"})

	require.True(t, ok)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, selectors.Label.Empty())
	assert.True(t, selectors.Field.Empty())
	assert.Empty(t, selectors.NamePrefix)
}

func TestParseSelectors_Parsed(t *testing.T) {
	t.Parallel()

	selectors, ok, _ := parseSelectorsFor(t,
		"labelSelector=env%3Dprod&fieldSelector=spec.platform%3Dvm&name=web-",
		[]string{"spec.platform"})

	require.True(t, ok)
	assert.Equal(t, selector.LabelSelector{
		{Key: "env", Operator: selector.OpEquals, Values: []string{"prod"}},
	}, selectors.Label)
	assert.Equal(t, selector.FieldSelector{
		{Field: "spec.platform", Operator: selector.OpEquals, Value: "vm"},
	}, selectors.Field)
	assert.Equal(t, "web-", selectors.NamePrefix)
}

func TestParseSelectors_MalformedLabelSelectorIsRejected(t *testing.T) {
	t.Parallel()

	_, ok, recorder := parseSelectorsFor(t, "labelSelector=env%3Dprod%2C%2C", []string{"spec.platform"})

	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "labelSelector")
}

// A field the resource does not support must be a 400 naming it — never a 200
// carrying the whole, unfiltered collection, which a client would mistake for a
// narrowed result.
func TestParseSelectors_UnsupportedFieldIsRejectedByName(t *testing.T) {
	t.Parallel()

	_, ok, recorder := parseSelectorsFor(t, "fieldSelector=spec.secret%3Dx", []string{"spec.platform"})

	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	body := recorder.Body.String()
	assert.Contains(t, body, "fieldSelector")
	assert.Contains(t, body, "spec.secret", "the response must name the rejected field")
	assert.Contains(t, body, "spec.platform", "the response must list what is supported")
}

func TestParseSelectors_AnyFieldIsRejectedWhenTheResourceSupportsNone(t *testing.T) {
	t.Parallel()

	_, ok, recorder := parseSelectorsFor(t, "fieldSelector=metadata.name%3Da", nil)

	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func parseSelectorsWithoutLabelsFor(
	t *testing.T, query string, allowed []string,
) (ginutil.Selectors, bool, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/roles?"+query, nil)

	selectors, ok := ginutil.ParseSelectorsWithoutLabels(ctx, allowed)

	return selectors, ok, recorder
}

// A resource with no label map answers a labelSelector with a 400 saying so. An
// empty page would be indistinguishable from "no role matches", which is a
// different and wrong answer.
func TestParseSelectorsWithoutLabels_RejectsALabelSelector(t *testing.T) {
	t.Parallel()

	_, ok, recorder := parseSelectorsWithoutLabelsFor(
		t, "labelSelector=env%3Dprod", []string{"spec.isBuiltIn"})

	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "labelSelector")
	assert.Contains(t, recorder.Body.String(), "no labels")
}

// The other two filters still work: a label-less resource is still named and
// still has fields.
func TestParseSelectorsWithoutLabels_AcceptsNameAndFields(t *testing.T) {
	t.Parallel()

	selectors, ok, recorder := parseSelectorsWithoutLabelsFor(
		t, "name=Admin&fieldSelector=spec.isBuiltIn%3Dtrue", []string{"spec.isBuiltIn"})

	require.True(t, ok)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "Admin", selectors.NamePrefix)
	assert.Equal(t, []string{"spec.isBuiltIn"}, selectors.Field.Fields())
	assert.True(t, selectors.Label.Empty())
}

func TestParseSelectorsWithoutLabels_StillRejectsAnUnsupportedField(t *testing.T) {
	t.Parallel()

	_, ok, recorder := parseSelectorsWithoutLabelsFor(
		t, "fieldSelector=spec.nope%3Dtrue", []string{"spec.isBuiltIn"})

	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "spec.nope")
}
