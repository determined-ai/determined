import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { ExperimentBase } from 'types';

const contextProvider = generateContext<RestApiState<ExperimentBase[]>>({
  initialState: {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  },
  name: 'ActiveExperiments',
});

export default contextProvider;
