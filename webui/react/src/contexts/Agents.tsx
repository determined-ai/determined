import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { Agent } from 'types';

const initialState = {
  errorCount: 0,
  hasLoaded: false,
  isLoading: false,
};

export default generateContext<RestApiState<Agent[]>>({
  initialState: initialState,
  name: 'Agents',
});
