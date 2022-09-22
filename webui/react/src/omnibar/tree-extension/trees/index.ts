import { activeRunStates, terminalCommandStates, terminalRunStatesKeys } from 'constants/states';
import { displayHelp, parseIds, visitAction } from 'omnibar/tree-extension/trees/actions';
import dev from 'omnibar/tree-extension/trees/dev';
import locations from 'omnibar/tree-extension/trees/goto';
import { Children, LeafNode, NonLeafNode } from 'omnibar/tree-extension/types';
import { paths } from 'routes/utils';
import {
  activateExperiment,
  archiveExperiment,
  getExperiments,
  getJupyterLabs,
  getTensorBoards,
  killExperiment,
  killJupyterLab,
  killTensorBoard,
  openOrCreateTensorBoard,
  pauseExperiment,
} from 'services/api';
import { launchJupyterLab } from 'utils/jupyter';

const root: NonLeafNode = {
  options: [
    {
      options: async (): Promise<Children> => {
        const { experiments: exps } = await getExperiments({
          orderBy: 'ORDER_BY_DESC',
          sortBy: 'SORT_BY_START_TIME',
          states: activeRunStates,
        });
        const options: LeafNode[] = exps.map((exp) => ({
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
          states: ['STATE_PAUSED'],
        });
        const options: LeafNode[] = exps.map((exp) => ({
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
          states: terminalRunStatesKeys.map((key) => `STATE_${key}` as const),
        });
        const options: Children = exps.map((exp) => ({
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
      aliases: ['stop', 'cancel'],
      options: [
        {
          options: async (): Promise<Children> => {
            const cmds = await getJupyterLabs({
              orderBy: 'ORDER_BY_DESC',
              sortBy: 'SORT_BY_START_TIME',
            });

            const options: LeafNode[] = cmds
              .filter((cmd) => !terminalCommandStates.has(cmd.state))
              .map((cmd) => ({
                onAction: () => killJupyterLab({ commandId: cmd.id }),
                title: `${cmd.name}`, // differentiate view only vs command text?
              }));
            return options;
          },
          title: 'jupyterLab',
        },
        {
          options: async (): Promise<Children> => {
            const cmds = await getTensorBoards({
              orderBy: 'ORDER_BY_DESC',
              sortBy: 'SORT_BY_START_TIME',
            });

            const options: LeafNode[] = cmds
              .filter((cmd) => !terminalCommandStates.has(cmd.state))
              .map((cmd) => ({
                onAction: () => killTensorBoard({ commandId: cmd.id }),
                title: `${cmd.name}`,
              }));
            return options;
          },
          title: 'tensorBoard',
        },
        {
          label: 'experiement <id>',
          options: async (): Promise<Children> => {
            const { experiments: exps } = await getExperiments({
              orderBy: 'ORDER_BY_DESC',
              sortBy: 'SORT_BY_START_TIME',
              states: activeRunStates,
            });
            const options: LeafNode[] = exps.map((exp) => ({
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
      aliases: ['open', 'create'],

      options: [
        {
          options: [
            {
              label: 'fromTrials <id1,id2,..>',
              onCustomInput: (inp: string): Children => {
                return [
                  {
                    onAction: () => {
                      openOrCreateTensorBoard({ trialIds: parseIds(inp) });
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
                      openOrCreateTensorBoard({ experimentIds: parseIds(inp) });
                    },
                    title: inp,
                  },
                ];
              },
              title: 'fromExperiments',
            },
          ],
          title: 'tensorBoard',
        },
        {
          options: [
            {
              onAction: () => launchJupyterLab({ slots: 0 }),
              title: 'zeroSlot',
            },
            {
              onAction: () => launchJupyterLab({ slots: 1 }),
              title: 'oneSlot',
            },
          ],
          title: 'jupyterLab',
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
