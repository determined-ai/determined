import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { User } from 'types';

const contextProvider = generateContext<RestApiState<User[]>>({
  initialState: {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  },
  name: 'Users',
});

export default contextProvider;
