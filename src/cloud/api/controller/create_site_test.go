package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"pixielabs.ai/pixielabs/src/cloud/api/controller"
	"pixielabs.ai/pixielabs/src/cloud/api/controller/testutils"
	authpb "pixielabs.ai/pixielabs/src/cloud/auth/proto"
	"pixielabs.ai/pixielabs/src/cloud/site_manager/sitemanagerpb"
	uuidpb "pixielabs.ai/pixielabs/src/common/uuid/proto"
	"pixielabs.ai/pixielabs/src/shared/services/handler"
	"pixielabs.ai/pixielabs/src/utils"
	"pixielabs.ai/pixielabs/src/utils/testingutils"
)

func TestCreateSiteHandler(t *testing.T) {
	orgID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	userID := "7ba7b810-9dad-11d1-80b4-00c04fd430c8"

	env, mockAuthClient, mockSiteManagerClient, _, _, _, cleanup := testutils.CreateTestAPIEnv(t)
	defer cleanup()

	req, err := http.NewRequest("POST", "/api/create-site",
		strings.NewReader("{\"accessToken\": \"the-token\", \"userEmail\": \"abc@hulu.com\", \"siteName\": \"def\"}"))
	assert.Nil(t, err)

	expectedAuthServiceReq := &authpb.CreateUserOrgRequest{
		AccessToken: "the-token",
		UserEmail:   "abc@hulu.com",
	}
	testReplyToken := testingutils.GenerateTestJWTToken(t, "jwt-key")
	testTokenExpiry := time.Now().Add(1 * time.Minute).Unix()
	createUserOrgReply := &authpb.CreateUserOrgResponse{
		Token:     testReplyToken,
		ExpiresAt: testTokenExpiry,
		OrgID:     &uuidpb.UUID{Data: []byte(orgID)},
		UserID:    &uuidpb.UUID{Data: []byte(userID)},
		UserInfo: &authpb.UserInfo{
			UserID:    utils.ProtoFromUUIDStrOrNil(userID),
			FirstName: "first",
			LastName:  "last",
			Email:     "abc@hulu.com",
		},
	}
	mockAuthClient.EXPECT().CreateUserOrg(gomock.Any(), expectedAuthServiceReq).Do(func(ctx context.Context, in *authpb.CreateUserOrgRequest) {
		assert.Equal(t, "the-token", in.AccessToken)
	}).Return(createUserOrgReply, nil)

	expectedSiteManagerReq := &sitemanagerpb.RegisterSiteRequest{
		SiteName: "def",
		OrgID:    &uuidpb.UUID{Data: []byte(orgID)},
	}
	registerSiteResponse := &sitemanagerpb.RegisterSiteResponse{
		SiteRegistered: true,
	}
	mockSiteManagerClient.EXPECT().RegisterSite(gomock.Any(), expectedSiteManagerReq).Return(registerSiteResponse, nil)

	rr := httptest.NewRecorder()
	h := handler.New(env, controller.CreateSiteHandler)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var parsedResponse struct {
		Token     string
		ExpiresAt int64
	}
	err = json.NewDecoder(rr.Body).Decode(&parsedResponse)
	assert.Nil(t, err)
	assert.Equal(t, testReplyToken, parsedResponse.Token)
	assert.Equal(t, testTokenExpiry, parsedResponse.ExpiresAt)

	// Check the token in the cookie.
	rawCookies := rr.Header().Get("Set-Cookie")
	header := http.Header{}
	header.Add("Cookie", rawCookies)
	req2 := http.Request{Header: header}
	sess, err := controller.GetDefaultSession(env, &req2)
	assert.Equal(t, testReplyToken, sess.Values["_at"])
	assert.Equal(t, "def", sess.Values["_auth_site"])
}

func TestCreateSiteHandler_IndividualSite(t *testing.T) {
	orgID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	userID := "7ba7b810-9dad-11d1-80b4-00c04fd430c8"

	env, mockAuthClient, mockSiteManagerClient, _, _, _, cleanup := testutils.CreateTestAPIEnv(t)
	defer cleanup()

	req, err := http.NewRequest("POST", "/api/create-site",
		strings.NewReader("{\"accessToken\": \"the-token\", \"userEmail\": \"abc@gmail.com\", \"siteName\": \"def\"}"))
	assert.Nil(t, err)

	expectedAuthServiceReq := &authpb.CreateUserOrgRequest{
		AccessToken: "the-token",
		UserEmail:   "abc@gmail.com",
	}
	testReplyToken := testingutils.GenerateTestJWTToken(t, "jwt-key")
	testTokenExpiry := time.Now().Add(1 * time.Minute).Unix()
	createUserOrgReply := &authpb.CreateUserOrgResponse{
		Token:     testReplyToken,
		ExpiresAt: testTokenExpiry,
		OrgID:     &uuidpb.UUID{Data: []byte(orgID)},
		UserID:    &uuidpb.UUID{Data: []byte(userID)},
		UserInfo: &authpb.UserInfo{
			UserID:    utils.ProtoFromUUIDStrOrNil(userID),
			FirstName: "first",
			LastName:  "last",
			Email:     "abc@hulu.com",
		},
	}
	mockAuthClient.EXPECT().CreateUserOrg(gomock.Any(), expectedAuthServiceReq).Do(func(ctx context.Context, in *authpb.CreateUserOrgRequest) {
		assert.Equal(t, "the-token", in.AccessToken)
	}).Return(createUserOrgReply, nil)

	expectedSiteManagerReq := &sitemanagerpb.RegisterSiteRequest{
		SiteName: "def",
		OrgID:    &uuidpb.UUID{Data: []byte(orgID)},
	}
	registerSiteResponse := &sitemanagerpb.RegisterSiteResponse{
		SiteRegistered: true,
	}
	mockSiteManagerClient.EXPECT().RegisterSite(gomock.Any(), expectedSiteManagerReq).Return(registerSiteResponse, nil)

	rr := httptest.NewRecorder()
	h := handler.New(env, controller.CreateSiteHandler)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var parsedResponse struct {
		Token     string
		ExpiresAt int64
	}
	err = json.NewDecoder(rr.Body).Decode(&parsedResponse)
	assert.Nil(t, err)
	assert.Equal(t, testReplyToken, parsedResponse.Token)
	assert.Equal(t, testTokenExpiry, parsedResponse.ExpiresAt)

	// Check the token in the cookie.
	rawCookies := rr.Header().Get("Set-Cookie")
	header := http.Header{}
	header.Add("Cookie", rawCookies)
	req2 := http.Request{Header: header}
	sess, err := controller.GetDefaultSession(env, &req2)
	assert.Equal(t, testReplyToken, sess.Values["_at"])
	assert.Equal(t, "def", sess.Values["_auth_site"])
}

func TestCreateSiteHandler_BadMethod(t *testing.T) {
	env, _, _, _, _, _, cleanup := testutils.CreateTestAPIEnv(t)
	defer cleanup()
	req, err := http.NewRequest("GET", "/api/create-site", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	h := handler.New(env, controller.CreateSiteHandler)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestCreateSiteHandler_UserCreationError(t *testing.T) {
	env, mockAuthClient, _, _, _, _, cleanup := testutils.CreateTestAPIEnv(t)
	defer cleanup()

	req, err := http.NewRequest("POST", "/api/create-site",
		strings.NewReader("{\"accessToken\": \"the-token\", \"userEmail\": \"abc@hulu.com\", \"siteName\": \"def\"}"))
	assert.Nil(t, err)

	expectedAuthServiceReq := &authpb.CreateUserOrgRequest{
		AccessToken: "the-token",
		UserEmail:   "abc@hulu.com",
	}

	mockAuthClient.EXPECT().CreateUserOrg(gomock.Any(), expectedAuthServiceReq).Do(func(ctx context.Context, in *authpb.CreateUserOrgRequest) {
		assert.Equal(t, "the-token", in.AccessToken)
	}).Return(nil, errors.New("could not create user"))

	rr := httptest.NewRecorder()
	h := handler.New(env, controller.CreateSiteHandler)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestCreateSiteHandler_SiteCreationError(t *testing.T) {
	orgID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	userID := "7ba7b810-9dad-11d1-80b4-00c04fd430c8"

	env, mockAuthClient, mockSiteManagerClient, _, _, _, cleanup := testutils.CreateTestAPIEnv(t)
	defer cleanup()

	req, err := http.NewRequest("POST", "/api/create-site",
		strings.NewReader("{\"accessToken\": \"the-token\", \"userEmail\": \"abc@hulu.com\", \"siteName\": \"def\"}"))
	assert.Nil(t, err)

	expectedAuthServiceReq := &authpb.CreateUserOrgRequest{
		AccessToken: "the-token",
		UserEmail:   "abc@hulu.com",
	}
	testReplyToken := testingutils.GenerateTestJWTToken(t, "jwt-key")
	testTokenExpiry := time.Now().Add(1 * time.Minute).Unix()
	createUserOrgReply := &authpb.CreateUserOrgResponse{
		Token:     testReplyToken,
		ExpiresAt: testTokenExpiry,
		OrgID:     &uuidpb.UUID{Data: []byte(orgID)},
		UserID:    &uuidpb.UUID{Data: []byte(userID)},
		UserInfo: &authpb.UserInfo{
			UserID:    utils.ProtoFromUUIDStrOrNil(userID),
			FirstName: "first",
			LastName:  "last",
			Email:     "abc@hulu.com",
		},
	}
	mockAuthClient.EXPECT().CreateUserOrg(gomock.Any(), expectedAuthServiceReq).Do(func(ctx context.Context, in *authpb.CreateUserOrgRequest) {
		assert.Equal(t, "the-token", in.AccessToken)
	}).Return(createUserOrgReply, nil)

	expectedSiteManagerReq := &sitemanagerpb.RegisterSiteRequest{
		SiteName: "def",
		OrgID:    &uuidpb.UUID{Data: []byte(orgID)},
	}
	mockSiteManagerClient.EXPECT().RegisterSite(gomock.Any(), expectedSiteManagerReq).Return(nil, errors.New("Could not create site"))

	rr := httptest.NewRecorder()
	h := handler.New(env, controller.CreateSiteHandler)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestCreateSiteHandler_SiteCreationFailed(t *testing.T) {
	orgID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	userID := "7ba7b810-9dad-11d1-80b4-00c04fd430c8"

	env, mockAuthClient, mockSiteManagerClient, _, _, _, cleanup := testutils.CreateTestAPIEnv(t)
	defer cleanup()

	req, err := http.NewRequest("POST", "/api/create-site",
		strings.NewReader("{\"accessToken\": \"the-token\", \"userEmail\": \"abc@hulu.com\", \"siteName\": \"def\"}"))
	assert.Nil(t, err)

	expectedAuthServiceReq := &authpb.CreateUserOrgRequest{
		AccessToken: "the-token",
		UserEmail:   "abc@hulu.com",
	}
	testReplyToken := testingutils.GenerateTestJWTToken(t, "jwt-key")
	testTokenExpiry := time.Now().Add(1 * time.Minute).Unix()
	createUserOrgReply := &authpb.CreateUserOrgResponse{
		Token:     testReplyToken,
		ExpiresAt: testTokenExpiry,
		OrgID:     &uuidpb.UUID{Data: []byte(orgID)},
		UserID:    &uuidpb.UUID{Data: []byte(userID)},
		UserInfo: &authpb.UserInfo{
			UserID:    utils.ProtoFromUUIDStrOrNil(userID),
			FirstName: "first",
			LastName:  "last",
			Email:     "abc@hulu.com",
		},
	}
	mockAuthClient.EXPECT().CreateUserOrg(gomock.Any(), expectedAuthServiceReq).Do(func(ctx context.Context, in *authpb.CreateUserOrgRequest) {
		assert.Equal(t, "the-token", in.AccessToken)
	}).Return(createUserOrgReply, nil)

	expectedSiteManagerReq := &sitemanagerpb.RegisterSiteRequest{
		SiteName: "def",
		OrgID:    &uuidpb.UUID{Data: []byte(orgID)},
	}
	registerSiteResponse := &sitemanagerpb.RegisterSiteResponse{
		SiteRegistered: false,
	}
	mockSiteManagerClient.EXPECT().RegisterSite(gomock.Any(), expectedSiteManagerReq).Return(registerSiteResponse, nil)

	rr := httptest.NewRecorder()
	h := handler.New(env, controller.CreateSiteHandler)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
