import { displayHelp, parseIds, visitAction } from 'omnibar/trees/actions';
import dev from 'omnibar/trees/dev';
import locations from 'omnibar/trees/goto';
import { Children, LeafNode, NLNode } from 'omnibar/types';
import { paths } from 'routes/utils';
import { activateExperiment, archiveExperiment, getExperiments, getNotebooks, getTensorboards,
  killExperiment, killNotebook, killTensorboard, openOrCreateTensorboard,
  pauseExperiment } from 'services/api';
import { launchNotebook } from 'utils/task';
import { activeRunStates, terminalCommandStates, terminalRunStates } from 'utils/types';

const root: NLNode = {
  options: [
    {
      options: async (): Promise<Children> => {
        const { experiments: exps } = await getExperiments({
          orderBy: 'ORDER_BY_DESC',
          sortBy: 'SORT_BY_START_TIME',
          states: activeRunStates,
        });
        const options: LeafNode[] = exps.map(exp => (
          {
            onAction: () => pauseExperiment({ experimentId: exp.id }),
            title: `${exp.id}`,
          }));
        return options;
      },
      title: 'pauseExperiment',
    },
    {
      options: async (): Promise<Children> => {
        const { experiments: exps } = await getExperiments({
          orderBy: 'ORDER_BY_DESC',
          sortBy: 'SORT_BY_START_TIME',
          states: [ 'STATE_PAUSED' ],
        });
        const options: LeafNode[] = exps.map(exp => (
          {
            onAction: () => activateExperiment({ experimentId: exp.id }),
            title: `${exp.id}`,
          }));
        return options;
      },
      title: 'activateExperiment',
    },
    {
      options: async (): Promise<Children> => {
        const { experiments: exps } = await getExperiments({
          orderBy: 'ORDER_BY_DESC',
          sortBy: 'SORT_BY_END_TIME',
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          states: Array.from(terminalRunStates).map(s => 'STATE_' + s) as any,
        });
        const options: Children = exps.map(exp => (
          {
            onAction: (): unknown => archiveExperiment({ experimentId: exp.id }),
            title: `${exp.id}`,
          }));
        return options;
      },
      title: 'archiveExperiment',
    },
    {
      options: locations,
      title: 'goto',
    },
    {
      aliases: [ 'stop', 'cancel' ],
      options: [
        {
          options: async (): Promise<Children> => {
            const cmds = await getNotebooks({
              orderBy: 'ORDER_BY_DESC',
              sortBy: 'SORT_BY_START_TIME',
            });

            const options: LeafNode[] = cmds
              .filter(cmd => !terminalCommandStates.has(cmd.state))
              .map(cmd => (
                {
                  onAction: () => killNotebook({ commandId: cmd.id }),
                  title: `${cmd.name}`, // differentiate view only vs command text?
                }));
            return options;
          },
          title: 'notebook',
        },
        {
          options: async (): Promise<Children> => {
            const cmds = await getTensorboards({
              orderBy: 'ORDER_BY_DESC',
              sortBy: 'SORT_BY_START_TIME',
            });

            const options: LeafNode[] = cmds
              .filter(cmd => !terminalCommandStates.has(cmd.state))
              .map(cmd => (
                {
                  onAction: () => killTensorboard({ commandId: cmd.id }),
                  title: `${cmd.name}`,
                }));
            return options;
          },
          title: 'tensorboard',
        },
        {
          label: 'experiement <id>',
          options: async (): Promise<Children> => {
            const { experiments: exps } = await getExperiments({
              orderBy: 'ORDER_BY_DESC',
              sortBy: 'SORT_BY_START_TIME',
              states: activeRunStates,
            });
            const options: LeafNode[] = exps.map(exp => (
              {
                onAction: () => killExperiment({ experimentId: exp.id }),
                title: `${exp.id}`,
              }));
            return options;
          },
          title: 'experiment',
        },

      ],
      title: 'kill', // stop sounds non-terminal...
    },
    {
      aliases: [ 'open', 'create' ],

      options: [
        {
          options: [
            {
              label: 'fromTrials <id1,id2,..>',
              onCustomInput: (inp: string): Children => {
                return [
                  {
                    onAction: () => {
                      openOrCreateTensorboard({ trialIds: parseIds(inp) });
                    },
                    title: inp,
                  },
                ];
              },
              title: 'fromTrials',
            },
            {
              label: 'fromExperiment <id1,id2,..>',
              onCustomInput: (inp: string): Children => {
                return [
                  {
                    onAction: () => {
                      openOrCreateTensorboard({ experimentIds: parseIds(inp) });
                    },
                    title: inp,
                  },
                ];
              },
              title: 'fromExperiments',
            },
          ],
          title: 'tensorboard',
        },
        {
          options: [
            {
              onAction: () => launchNotebook(0),
              title: 'zeroSlot',
            },
            {
              onAction: () => launchNotebook(1),
              title: 'oneSlot',
            },
          ],
          title: 'notebook',
        },
      ],
      title: 'launch',
    },
    {
      onAction: visitAction(paths.logout()),
      title: 'logout',
    },
    {
      options: dev,
      title: 'dev',
    },
    {
      onAction: displayHelp,
      title: 'help',
    },
  ],
  title: 'root',
};

export default root;
