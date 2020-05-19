import { generateContext } from 'contexts';
import { Navigation } from 'types';

const contextProvider = generateContext<Navigation>({
  initialState:  {
    showNavBar: true,
    showSideBar: true,
  },
  name: 'Navigation',
});

export default contextProvider;
