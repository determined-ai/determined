import { visitAction } from 'omnibar/actions';
import { Children, TreeNode } from 'omnibar/types';
import { paths } from 'routes/utils';
import { getExperiments, getExpTrials } from 'services/api';
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
    options: async (): Promise<Children> => {
      const { experiments: exps } = await getExperiments(
        { limit: 1, orderBy: 'ORDER_BY_DESC', sortBy: 'SORT_BY_ID' },
      );
      const options = new Array(exps[0].id).fill(0).map((_, idx) => {
        return {
          onAction: visitAction(paths.experimentDetails(idx+1)),
          title: `${idx+1}`, // TODO render more info?
        };
      });
      return options;
    },
    title: 'experiment',
  },
  {
    options: async (): Promise<Children> => {
      const { experiments: exps } = await getExperiments(
        { limit: 1, orderBy: 'ORDER_BY_DESC', sortBy: 'SORT_BY_ID' },
      );

      const { trials } = await getExpTrials(
        { id: exps[0].id, limit: 1, orderBy: 'ORDER_BY_DESC', sortBy: 'SORT_BY_ID' },
      );

      const lastId = trials[0].id;
      const options = new Array(lastId).fill(0).map((_, idx) => {
        return {
          onAction: visitAction(paths.trialDetails(idx+1)),
          title: `${idx+1}`,
        };
      });

      return options;
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
