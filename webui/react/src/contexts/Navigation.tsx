import { generateContext } from 'contexts';
import { Navigation } from 'types';

const contextProvider = generateContext<Navigation>({
  initialState:  { showChrome: true },
  name: 'Navigation',
});

export default contextProvider;
