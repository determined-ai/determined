import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { ExperimentOld } from 'types';

const contextProvider = generateContext<RestApiState<ExperimentOld[]>>({
  initialState: {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  },
  name: 'ActiveExperiments',
});

export default contextProvider;
