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

	selectors, ok := ginutil.ParseSelectors(ctx, ginutil.LabelMetadataSelector, allowed)

	return selectors, ok, recorder
}

func TestParseSelectors_Empty(t *testing.T) {
	t.Parallel()

	selectors, ok, recorder := parseSelectorsFor(t, "", []string{"spec.platform"})

	require.True(t, ok)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, selectors.Metadata.Empty())
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
	}, selectors.Metadata)
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

// parseSelectorsAs runs the parser for a resource of the given metadata kind.
func parseSelectorsAs(
	t *testing.T, metadata ginutil.MetadataSelector, query string, allowed []string,
) (ginutil.Selectors, bool, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/agents?"+query, nil)

	selectors, ok := ginutil.ParseSelectors(ctx, metadata, allowed)

	return selectors, ok, recorder
}

// An agent's metadata is reported, not set, so it answers attributeSelector.
func TestParseSelectors_AttributeResourceAcceptsAttributeSelector(t *testing.T) {
	t.Parallel()

	selectors, ok, recorder := parseSelectorsAs(
		t, ginutil.AttributeMetadataSelector, "attributeSelector=os.type%3Dlinux", nil)

	require.True(t, ok)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.False(t, selectors.Metadata.Empty())
}

// Sending the wrong metadata parameter has to be an error rather than an unread
// query value: gin ignores what nothing reads, so a client narrowing a list the
// wrong way would otherwise get the whole one with a 200.
func TestParseSelectors_AttributeResourceRejectsLabelSelector(t *testing.T) {
	t.Parallel()

	_, ok, recorder := parseSelectorsAs(
		t, ginutil.AttributeMetadataSelector, "labelSelector=env%3Dprod", nil)

	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "labelSelector")
	assert.Contains(t, recorder.Body.String(), "attributeSelector")
}

func TestParseSelectors_LabelResourceRejectsAttributeSelector(t *testing.T) {
	t.Parallel()

	_, ok, recorder := parseSelectorsAs(
		t, ginutil.LabelMetadataSelector, "attributeSelector=os.type%3Dlinux", []string{"spec.platform"})

	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "attributeSelector")
	assert.Contains(t, recorder.Body.String(), "labelSelector")
}

// A resource with neither kind rejects both, rather than answering one of them
// against an empty map.
func TestParseSelectors_NoMetadataResourceRejectsBoth(t *testing.T) {
	t.Parallel()

	for _, query := range []string{"labelSelector=env%3Dprod", "attributeSelector=env%3Dprod"} {
		_, ok, recorder := parseSelectorsAs(
			t, ginutil.NoMetadataSelector, query, []string{"spec.isBuiltIn"})

		require.False(t, ok, query)
		assert.Equal(t, http.StatusBadRequest, recorder.Code, query)
		assert.Contains(t, recorder.Body.String(), "no labels or attributes", query)
	}
}

func parseSelectorsWithoutLabelsFor(
	t *testing.T, query string, allowed []string,
) (ginutil.Selectors, bool, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/roles?"+query, nil)

	selectors, ok := ginutil.ParseSelectors(ctx, ginutil.NoMetadataSelector, allowed)

	return selectors, ok, recorder
}

// A resource with neither kind of metadata answers a labelSelector with a 400
// saying so. An empty page would be indistinguishable from "no role matches",
// which is a different and wrong answer.
func TestParseSelectors_NoMetadataResourceNamesWhatItLacks(t *testing.T) {
	t.Parallel()

	_, ok, recorder := parseSelectorsWithoutLabelsFor(
		t, "labelSelector=env%3Dprod", []string{"spec.isBuiltIn"})

	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "labelSelector")
	assert.Contains(t, recorder.Body.String(), "no labels or attributes")
}

// The other two filters still work: a resource with no metadata map is still
// named and still has fields.
func TestParseSelectors_NoMetadataResourceStillFiltersByNameAndFields(t *testing.T) {
	t.Parallel()

	selectors, ok, recorder := parseSelectorsWithoutLabelsFor(
		t, "name=Admin&fieldSelector=spec.isBuiltIn%3Dtrue", []string{"spec.isBuiltIn"})

	require.True(t, ok)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "Admin", selectors.NamePrefix)
	assert.Equal(t, []string{"spec.isBuiltIn"}, selectors.Field.Fields())
	assert.True(t, selectors.Metadata.Empty())
}

func TestParseSelectors_NoMetadataResourceStillRejectsAnUnsupportedField(t *testing.T) {
	t.Parallel()

	_, ok, recorder := parseSelectorsWithoutLabelsFor(
		t, "fieldSelector=spec.nope%3Dtrue", []string{"spec.isBuiltIn"})

	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "spec.nope")
}

func TestParseSelectors_NameContains(t *testing.T) {
	t.Parallel()

	selectors, ok, recorder := parseSelectorsFor(t, "nameContains=coll", []string{"spec.platform"})

	require.True(t, ok)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "coll", selectors.NameContains)
	assert.Empty(t, selectors.NamePrefix)
}

// The two name filters are independent and combine, so a caller can anchor a
// scan to an indexed prefix.
func TestParseSelectors_NamePrefixAndContainsCombine(t *testing.T) {
	t.Parallel()

	selectors, ok, _ := parseSelectorsFor(t, "name=otel-&nameContains=coll", []string{"spec.platform"})

	require.True(t, ok)
	assert.Equal(t, "otel-", selectors.NamePrefix)
	assert.Equal(t, "coll", selectors.NameContains)
}
