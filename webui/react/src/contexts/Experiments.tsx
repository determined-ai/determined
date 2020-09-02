import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { ExperimentX } from 'types';

const contextProvider = generateContext<RestApiState<ExperimentX[]>>({
  initialState: {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  },
  name: 'Experiments',
});

export default contextProvider;
