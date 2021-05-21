import { alertAction, parseIds, visitAction } from 'omnibar/tree-extension/trees/actions';
import { Children, TreeNode } from 'omnibar/tree-extension/types';
import { paths } from 'routes/utils';
import { getExperimentDetails, getTrialDetails } from 'services/api';
import { getNotebooks, getTensorboards } from 'services/api';
import { terminalCommandStates } from 'utils/types';
import { openCommand } from 'wait';

const locations: TreeNode[] = [
  {
    onAction: visitAction(paths.experimentList()),
    title: 'experiments',
  },
  {
    aliases: [ 'agents', 'resourcePools' ],
    onAction: visitAction(paths.cluster()),
    title: 'cluster',
  },
  {
    label: 'experiment <id>',
    onCustomInput: async (inp: string): Promise<Children> => {
      let expExists = false;
      const id = parseIds(inp)[0];
      try {
        await getExperimentDetails({ id });
        expExists = true;
      } catch {
      }

      const onAction = expExists ? visitAction(paths.experimentDetails(id)) :
        alertAction(`Invalid experiment ID ${id}`);

      // TODO we could generate this `<id>` arg label and the label for the
      // parent together instead of separately.
      const label = inp === '' ? '<id>' : expExists ? inp : `${inp} (doesn't exist)`;
      return [
        { label, onAction, title: inp },
      ];
    },
    title: 'experiment',
  },
  {
    onCustomInput: (inp: string): Children => {

      const onAction = async () => {
        const id = parseIds(inp)[0];
        try {
          const trial = await getTrialDetails({ id });
          visitAction(paths.trialDetails(trial.id, trial.experimentId))();
        } catch {
          alertAction(`Invalid trial ID ${id}`);
        }
      };

      // TODO we could generate this `<id>` arg label and the label for the
      // parent together instead of separately.
      const label = inp === '' ? '<id>' : inp;
      return [
        { label, onAction, title: inp },
      ];
    },
    title: 'trial',
  },
  {
    onAction: visitAction(paths.dashboard()),
    title: 'dashboard',
  },
  {
    aliases: [ 'notebooks', 'tensorboards', 'commands', 'shells' ],
    onAction: visitAction(paths.taskList()),
    title: 'tasks',
  },
  {
    options: async (): Promise<Children> => {
      const tsbs = await getTensorboards({});
      return tsbs
        .filter(tsb => !terminalCommandStates.has(tsb.state))
        .map(tsb => ({
          onAction: () => openCommand(tsb),
          title: `${JSON.stringify(tsb.misc)}`,
        }));
    },
    title: 'tensorboard',
  },
  {
    options: async (): Promise<Children> => {
      const nbs = await getNotebooks({});
      return nbs
        .filter(nb => !terminalCommandStates.has(nb.state))
        .map(nb => ({
          onAction: () => openCommand(nb),
          title: nb.name,
        }));
    },
    title: 'notebook',
  },
  {
    onAction: visitAction(paths.masterLogs()),
    title: 'masterLogs',
  },
];

export default locations;
