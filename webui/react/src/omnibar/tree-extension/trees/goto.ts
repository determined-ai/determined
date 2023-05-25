import { terminalCommandStates } from 'constants/states';
import { alertAction, parseIds, visitAction } from 'omnibar/tree-extension/trees/actions';
import { Children, TreeNode } from 'omnibar/tree-extension/types';
import { paths } from 'routes/utils';
import { getExperimentDetails, getTrialDetails } from 'services/api';
import { getJupyterLabs, getTensorBoards } from 'services/api';
import { openCommand } from 'utils/wait';

const locations: TreeNode[] = [
  {
    aliases: ['agents', 'resourcePools'],
    onAction: visitAction(paths.clusters()),
    title: 'cluster',
  },
  {
    label: 'experiment <id>',
    onCustomInput: (inp: string): Children => {
      const onAction = async () => {
        const id = parseIds(inp)[0];
        try {
          await getExperimentDetails({ id });
          visitAction(paths.experimentDetails(id))();
        } catch {
          alertAction(`Invalid experiment ID ${id}`)();
        }
      };

      const label = inp === '' ? '<id>' : inp;
      return [{ label, onAction, title: inp }];
    },
    title: 'experiment',
  },
  {
    label: 'trial <id>',
    onCustomInput: (inp: string): Children => {
      const onAction = async () => {
        const id = parseIds(inp)[0];
        try {
          const trial = await getTrialDetails({ id });
          visitAction(paths.trialDetails(trial.id, trial.experimentId))();
        } catch {
          alertAction(`Invalid trial ID ${id}`)();
        }
      };

      // we could generate this `<id>` arg label and the label for the
      // parent node together instead of separately.
      const label = inp === '' ? '<id>' : inp;
      return [{ label, onAction, title: inp }];
    },
    title: 'trial',
  },
  {
    aliases: ['jupyterLabs', 'tensorBoards', 'commands', 'shells'],
    onAction: visitAction(paths.taskList()),
    title: 'tasks',
  },
  {
    options: async (): Promise<Children> => {
      const tsbs = await getTensorBoards({});
      return tsbs
        .filter((tsb) => !terminalCommandStates.has(tsb.state))
        .map((tsb) => ({
          onAction: () => openCommand(tsb),
          title: `${JSON.stringify(tsb.misc)}`,
        }));
    },
    title: 'tensorBoard',
  },
  {
    options: async (): Promise<Children> => {
      const nbs = await getJupyterLabs({});
      return nbs
        .filter((nb) => !terminalCommandStates.has(nb.state))
        .map((nb) => ({
          onAction: () => openCommand(nb),
          title: nb.name,
        }));
    },
    title: 'jupyterLab',
  },
  {
    onAction: visitAction(paths.clusterLogs()),
    title: 'clusterLogs',
  },
];

export default locations;
