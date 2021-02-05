import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { ExperimentPagination } from 'types';

const contextProvider = generateContext<RestApiState<ExperimentPagination>>({
  initialState: {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  },
  name: 'ActiveExperiments',
});

export default contextProvider;
