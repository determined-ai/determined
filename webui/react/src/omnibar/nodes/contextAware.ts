import { Children, TreeNode } from 'omnibar/AsyncTree';
import { killExperiment } from 'services/api';
import { noOp } from 'services/utils';

// TODO contextual trees. (different branches based on a current state eg page)
const contextAware: TreeNode = {
  options: (): Children => {
    switch (findRouteId()) {
      case 'experimentDetails':
        // TODO get route params
        return experimentOptions(0);
      case 'trialDetails':
        // TODO get route params
        return trialOptions();
      default:
        return [];
    }
  },
  title: 'contextAware', // rename me. could be merged with the upper level
};

const experimentOptions = (id: number): TreeNode[] => {
  // TODO check state and offer options.
  return [
    {
      onAction: () => killExperiment({ experimentId: id }),
      title: 'kill',
    },
  ];
};

const trialOptions = (): TreeNode[] => {
  return [
    {
      onAction: noOp,
      title: 'openInTensorboard',
    },
  ];
};

const findRouteId = (): string | undefined => {
  // TODO reuse definitions in routes.
  const expDetailsRe = new RegExp(`${process.env.PUBLIC_URL}/experiments/d+/?`, 'i');
  if (expDetailsRe.test(window.location.href)) {
    return 'experimentDetails';
  }
};

export default contextAware;
