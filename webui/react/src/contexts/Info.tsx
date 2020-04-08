import { generateContext } from 'contexts';
import { DeterminedInfo } from 'types';

const contextProvider = generateContext<DeterminedInfo>({
  initialState: {
    clusterId: '',
    masterId: '',
    telemetry: { enabled: false },
    version: process.env.VERSION || '',
  },
  name: 'DeterminedInfo',
});

export default contextProvider;
