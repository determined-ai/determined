import { Children, LeafNode, NLNode } from 'AsyncTree';
import { archiveExperiment, getExperiments, killExperiment } from 'services/api';
import { activeRunStates, terminalRunStates } from 'utils/types';

const alertAction = (msg: string) => ((): void => alert(msg));
const visitAction = (url: string) => ((): void => window.location.assign(url));

const root: NLNode  = {
  options: [
    {
      onAction: (): void => alert(new Date()),
      title: 'showTime',
    },
    {
      options: async (): Promise<Children> => {
        const {experiments: exps} = await getExperiments({ states: activeRunStates });
        const options: LeafNode[] = exps.map(exp => (
          {
            onAction: (): unknown => killExperiment({ experimentId: exp.id }),
            title: `${exp.id}`,
          })); // is use of `this` discouraged?
        return options;
      },
      title: 'killExperiments',
    },
    {
      options: async (): Promise<Children> => {
        const {experiments: exps} = await getExperiments({ states: Array.from(terminalRunStates) as any });
        const options: Children = exps.map(exp => (
          {
            onAction: (): unknown => archiveExperiment({experimentId: exp.id}),
            title: `${exp.id}`,
          })); // is use of `this` discouraged?
        return options;
      },
      title: 'archiveExperiments',
    },
    {
      options: [
        {
          onAction: visitAction('/ui/experiments'),
          title: 'experiments',
        },
        {
          options: async (): Promise<Children> => {
            const {experiments: exps} = await getExperiments({});
            // const options: LeafNode[] = exps.map(exp => (
            //   {
            //     options: new Array(3).fill(null).map((_, idx) => idx+1),
            //     title: `${exp.id}`, // render more info
            //   })); // is use of `this` discouraged?
            const options: Children = exps.map(exp => (
              {
                onAction: visitAction('/ui/experiments/' + exp.id),
                title: `${exp.id}`, // render more info
              })); // is use of `this` discouraged?
            return options;

          },
          title: 'experiment',
        },
        {
          onAction: visitAction('/ui/experiments'),
          title: 'tensorboards',
        },
      ],
      title: 'goto',
    },
    {
      options: [
        {
          onAction: alertAction('created zeroslot notebook'),
          title: 'zeroSlot',
        },
        {
          onAction: alertAction('created oneslot notebook'),
          title: 'oneSlot',
        },
      ],
      title: 'launchNotebook',
    },
    {
      options: [
        {
          onAction: alertAction('restarted master'),
          title: 'restart',
        },
        {
          onAction: alertAction('reloaded master'),
          title: 'reload',
        },
        {
          onAction: alertAction('here are the logs..'),
          title: 'showlogs',
        },
      ],
      title: 'manageCluster',
    },
  ],
  title: 'root',
};

export default root;
