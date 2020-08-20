import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { DetailedUser } from 'types';

const contextProvider = generateContext<RestApiState<DetailedUser[]>>({
  initialState: {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  },
  name: 'Users',
});

export default contextProvider;
