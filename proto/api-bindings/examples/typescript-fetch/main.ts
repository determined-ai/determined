import * as Config from './configuration';
import * as Api from './api';

const SERVER_ADDRESS = 'http://localhost:8080';
const USERNAME = 'determined';
const PASSWORD = '';

const initialApiConfig = new Config.Configuration({basePath: SERVER_ADDRESS});
const detApi = {
  Auth: new Api.AuthenticationApi(initialApiConfig),
  Experiments: new Api.ExperimentsApi(initialApiConfig),
};

const updatedApiConfigParams = (apiConfig?: Config.ConfigurationParameters):
Config.ConfigurationParameters => {
  return {
    ...initialApiConfig,
    ...apiConfig,
  };
};

// Update references to generated API code with new configuration.
export const updateDetApi = (apiConfig: Config.ConfigurationParameters): void => {
  const config = updatedApiConfigParams(apiConfig);
  detApi.Auth = new Api.AuthenticationApi(config);
  detApi.Experiments = new Api.ExperimentsApi(config);
};

(async () => {
  // Login and set up the authentication key.
  const resposne = await detApi.Auth.determinedLogin({username: USERNAME, password: PASSWORD});
  updateDetApi({apiKey: 'Bearer ' + resposne.token})

  const curUser = await detApi.Auth.determinedCurrentUser();
  console.log(curUser)

  // const patchedExp = await detApi.Experiments.determinedPatchExperiment(1, {description: 'patched description'} as Api.V1Experiment)
  // console.log(patchedExp)

})()
