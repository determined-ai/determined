import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { ExperimentItem } from 'types';

const contextProvider = generateContext<RestApiState<ExperimentItem[]>>({
  initialState: {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  },
  name: 'Experiments',
});

export default contextProvider;
