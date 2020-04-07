import { generateContext } from 'contexts';
import { Agent } from 'types';

const contextProvider = generateContext<Agent[]>({
  initialState: [],
  name: 'Agent',
});

export default contextProvider;
