import { generateContext } from 'contexts';
import { ActionType, RestApiState } from 'hooks/useRestApi';
import { Experiment } from 'types';

const contextProvider = generateContext<RestApiState<Experiment[]>>({
  initialState: {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  },
  name: 'Experiments',
});

export default contextProvider;
