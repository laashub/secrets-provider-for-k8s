package k8s_secrets_storage

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/cyberark/cyberark-secrets-provider-for-k8s/pkg/log/messages"
	"github.com/cyberark/cyberark-secrets-provider-for-k8s/pkg/secrets/k8s_secrets_storage/mocks"
)

func TestProvideConjurSecrets(t *testing.T) {
	Convey("getVariableIDsToRetrieve", t, func() {

		Convey("Given a non-empty pathMap", func() {
			pathMap := make(map[string][]string)

			pathMap["account/var_path1"] = []string{"secret1:key1"}
			pathMap["account/var_path2"] = []string{"secret1:key2"}
			variableIDsExpected := []string{"account/var_path1", "account/var_path2"}
			variableIDsActual, err := getVariableIDsToRetrieve(pathMap)

			Convey("Finishes without raising an error", func() {
				So(err, ShouldEqual, nil)
			})

			Convey("Returns variable IDs in the pathMap as expected", func() {
				// Sort actual and expected, because output order can change
				sort.Strings(variableIDsActual)
				sort.Strings(variableIDsExpected)
				eq := reflect.DeepEqual(variableIDsActual, variableIDsExpected)
				So(eq, ShouldEqual, true)
			})
		})

		Convey("Given an empty pathMap", func() {
			pathMap := make(map[string][]string)

			Convey("Raises an error that the map input is empty", func() {
				_, err := getVariableIDsToRetrieve(pathMap)
				So(err.Error(), ShouldEqual, messages.CSPFK025E)
			})
		})
	})

	Convey("updateK8sSecretsMapWithConjurSecrets", t, func() {
		Convey("Given one K8s secret with one Conjur secret", func() {
			secret := []byte{'s', 'u', 'p', 'e', 'r'}
			conjurSecrets := make(map[string][]byte)
			conjurSecrets["account:variable:allowed/username"] = secret

			newDataEntriesMap := make(map[string][]byte)
			newDataEntriesMap["username"] = []byte("allowed/username")

			k8sSecretsMap := make(map[string]map[string][]byte)
			k8sSecretsMap["mysecret"] = newDataEntriesMap

			pathMap := make(map[string][]string)
			pathMap["allowed/username"] = []string{"mysecret:username"}

			k8sSecretsStruct := K8sSecretsMap{k8sSecretsMap, pathMap}
			err := updateK8sSecretsMapWithConjurSecrets(&k8sSecretsStruct, conjurSecrets)

			Convey("Finishes without raising an error", func() {
				So(err, ShouldEqual, nil)
			})

			Convey("Replaces secret variable IDs in k8sSecretsMap with their corresponding secret value", func() {
				eq := reflect.DeepEqual(k8sSecretsStruct.K8sSecrets["mysecret"]["username"], []byte{'s', 'u', 'p', 'e', 'r'})
				So(eq, ShouldEqual, true)
			})
		})

		Convey("Given 2 k8s secrets that need the same Conjur secret", func() {
			secret := []byte{'s', 'u', 'p', 'e', 'r'}
			conjurSecrets := make(map[string][]byte)
			conjurSecrets["account:variable:allowed/username"] = secret

			dataEntriesMap := make(map[string][]byte)
			dataEntriesMap["username"] = []byte("allowed/username")

			k8sSecretsMap := make(map[string]map[string][]byte)
			k8sSecretsMap["secret"] = dataEntriesMap
			k8sSecretsMap["another-secret"] = dataEntriesMap

			pathMap := make(map[string][]string)
			pathMap["allowed/username"] = []string{"secret:username", "another-secret:username"}

			k8sSecretsStruct := K8sSecretsMap{k8sSecretsMap, pathMap}
			err := updateK8sSecretsMapWithConjurSecrets(&k8sSecretsStruct, conjurSecrets)

			Convey("Finishes without raising an error", func() {
				So(err, ShouldEqual, nil)
			})

			Convey("Replaces both Variable IDs in k8sSecretsMap to their corresponding secret values without errors", func() {
				eq := reflect.DeepEqual(k8sSecretsStruct.K8sSecrets["secret"]["username"], secret)
				So(eq, ShouldEqual, true)

				eq = reflect.DeepEqual(k8sSecretsStruct.K8sSecrets["another-secret"]["username"], secret)
				So(eq, ShouldEqual, true)
			})
		})
	})

	Convey("RetrieveRequiredK8sSecrets", t, func() {
		mockK8sSecretsClient := &mocks.MockK8sSecretsClient{
			Permissions: k8sSecretsPermissions(true, true),
		}

		Convey("Given an existing k8s secret that is mapped to an existing conjur secret", func() {
			prepareMockDBs()

			addK8sSecretToMockDB("k8s_secret1", "conjur_variable1")
			requiredSecrets := []string{"k8s_secret1"}

			k8sSecretsMap, err := RetrieveRequiredK8sSecrets(mockK8sSecretsClient, "someNameSpace", requiredSecrets)

			Convey("Finishes without raising an error", func() {
				So(err, ShouldEqual, nil)
			})

			Convey("Creates K8sSecrets map as expected", func() {
				expectedK8sSecretsData := make(map[string][]byte)
				expectedK8sSecretsData["data_key"] = []byte("conjur_variable1")

				expectedK8sSecrets := make(map[string]map[string][]byte)
				expectedK8sSecrets["k8s_secret1"] = expectedK8sSecretsData

				eq := reflect.DeepEqual(k8sSecretsMap.K8sSecrets, expectedK8sSecrets)
				So(eq, ShouldEqual, true)
			})

			Convey("Creates PathMap map as expected", func() {
				expectedPathMap := make(map[string][]string)
				expectedPathMap["conjur_variable1"] = []string{fmt.Sprintf("%s:%s", "k8s_secret1", "data_key")}

				eq := reflect.DeepEqual(k8sSecretsMap.PathMap, expectedPathMap)
				So(eq, ShouldEqual, true)
			})
		})

		Convey("Given no 'get' permissions on the 'secrets' k8s resource", func() {
			prepareMockDBs()

			addK8sSecretToMockDB("k8s_secret1", "conjur_variable1")
			requiredSecrets := []string{"k8s_secret1"}

			mockK8sSecretsClient := &mocks.MockK8sSecretsClient{
				Permissions: k8sSecretsPermissions(false, true),
			}

			_, err := RetrieveRequiredK8sSecrets(mockK8sSecretsClient, "someNameSpace", requiredSecrets)

			Convey("raises proper error", func() {
				So(err.Error(), ShouldEqual, messages.CSPFK020E)
			})
		})

		Convey("Given a k8s secret without 'conjur-map'", func() {
			prepareMockDBs()

			secretData := make(map[string][]byte)
			secretData["not-conjur-map"] = []byte("some-data")
			mocks.MockK8sDB["no_conjur_map_secret"] = mocks.MockK8sSecret{
				Data: secretData,
			}

			requiredSecrets := []string{"no_conjur_map_secret"}

			_, err := RetrieveRequiredK8sSecrets(mockK8sSecretsClient, "someNameSpace", requiredSecrets)

			Convey("raises proper error", func() {
				So(err.Error(), ShouldEqual, fmt.Sprintf(messages.CSPFK028E, "no_conjur_map_secret"))
			})
		})
	})

	Convey("PatchRequiredK8sSecrets", t, func() {
		Convey("Given no 'patch' permissions on the 'secrets' k8s resource", func() {
			prepareMockDBs()

			addK8sSecretToMockDB("k8s_secret1", "conjur_variable1")
			requiredSecrets := []string{"k8s_secret1"}

			mockK8sSecretsClient := &mocks.MockK8sSecretsClient{
				Permissions: k8sSecretsPermissions(true, false),
			}

			k8sSecretsMap, err := RetrieveRequiredK8sSecrets(mockK8sSecretsClient, "someNameSpace", requiredSecrets)

			Convey("Finishes without raising an error", func() {
				So(err, ShouldEqual, nil)
			})

			err = PatchRequiredK8sSecrets(mockK8sSecretsClient, "someNameSpace", k8sSecretsMap)

			Convey("raises proper error", func() {
				So(err.Error(), ShouldEqual, messages.CSPFK022E)
			})
		})
	})

	Convey("run", t, func() {
		var mockAccessToken mocks.MockAccessToken
		var mockConjurSecretsRetriever mocks.MockConjurSecretsRetriever
		mockK8sSecretsClient := &mocks.MockK8sSecretsClient{
			Permissions: k8sSecretsPermissions(true, true),
		}

		Convey("Given 2 k8s secrets that only one is required by the pod", func() {
			prepareMockDBs()

			addK8sSecretToMockDB("k8s_secret1", "conjur_variable1")
			addK8sSecretToMockDB("k8s_secret2", "conjur_variable2")
			requiredSecrets := []string{"k8s_secret1"}

			err := run(
				mockK8sSecretsClient,
				"someNameSpace",
				requiredSecrets,
				mockAccessToken,
				mockConjurSecretsRetriever,
			)

			Convey("Finishes without raising an error", func() {
				So(err, ShouldEqual, nil)
			})

			Convey("Updates K8s secrets with their corresponding Conjur secrets", func() {
				verifyK8sSecretValue("k8s_secret1", "conjur_secret1")
			})

			Convey("Does not updates other K8s secrets", func() {
				actualK8sSecretDataValue := mocks.MockK8sDB["k8s_secret2"].Data["data_key"]
				So(actualK8sSecretDataValue, ShouldEqual, nil)
			})
		})

		Convey("Given 2 k8s secrets that are required by the pod - each one has its own Conjur secret", func() {
			prepareMockDBs()

			addK8sSecretToMockDB("k8s_secret1", "conjur_variable1")
			addK8sSecretToMockDB("k8s_secret2", "conjur_variable2")
			requiredSecrets := []string{"k8s_secret1", "k8s_secret2"}

			err := run(
				mockK8sSecretsClient,
				"someNameSpace",
				requiredSecrets,
				mockAccessToken,
				mockConjurSecretsRetriever,
			)

			Convey("Finishes without raising an error", func() {
				So(err, ShouldEqual, nil)
			})

			Convey("Updates K8s secrets with their corresponding Conjur secrets", func() {
				verifyK8sSecretValue("k8s_secret1", "conjur_secret1")
				verifyK8sSecretValue("k8s_secret2", "conjur_secret2")
			})
		})

		Convey("Given 2 k8s secrets that are required by the pod - both have the same Conjur secret", func() {
			prepareMockDBs()

			addK8sSecretToMockDB("k8s_secret1", "conjur_variable1")
			addK8sSecretToMockDB("k8s_secret2", "conjur_variable1")
			requiredSecrets := []string{"k8s_secret1", "k8s_secret2"}

			err := run(
				mockK8sSecretsClient,
				"someNameSpace",
				requiredSecrets,
				mockAccessToken,
				mockConjurSecretsRetriever,
			)

			Convey("Finishes without raising an error", func() {
				So(err, ShouldEqual, nil)
			})

			Convey("Updates K8s secrets with their corresponding Conjur secrets", func() {
				verifyK8sSecretValue("k8s_secret1", "conjur_secret1")
				verifyK8sSecretValue("k8s_secret2", "conjur_secret1")
			})
		})
	})
}

func prepareMockDBs() {
	mocks.MockK8sDB = make(map[string]mocks.MockK8sSecret)

	mocks.MockConjurDB = make(map[string][]byte)
	mocks.MockConjurDB["conjur_variable1"] = []byte("conjur_secret1")
	mocks.MockConjurDB["conjur_variable2"] = []byte("conjur_secret2")
}

func addK8sSecretToMockDB(secretName string, conjurVariable string) {
	secretDataEntries := make(map[string]string)
	secretDataEntries["data_key"] = conjurVariable
	mocks.MockK8sDB[secretName] = mocks.CreateMockK8sSecret(secretDataEntries)
}

func verifyK8sSecretValue(secretName string, value string) {
	actualK8sSecretDataValue := mocks.MockK8sDB[secretName].Data["data_key"]
	expectedK8sSecretDataValue := []byte(value)
	eq := reflect.DeepEqual(actualK8sSecretDataValue, expectedK8sSecretDataValue)
	So(eq, ShouldEqual, true)
}

func k8sSecretsPermissions(getPermission bool, patchPermission bool) map[string]bool {
	permissions := make(map[string]bool)
	permissions["get"] = getPermission
	permissions["patch"] = patchPermission

	return permissions
}
