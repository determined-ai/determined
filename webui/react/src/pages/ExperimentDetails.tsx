import { Breadcrumb, Button, Popconfirm, Space } from 'antd';
import React, { useCallback, useState } from 'react';
import { useParams } from 'react-router';

import ExperimentInfoBox from 'components/ExperimentInfoBox';
import Icon from 'components/Icon';
import Link from 'components/Link';
import Message from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { archiveExperiment, killExperiment, launchTensorboard, setExperimentState,
} from 'services/api';
import { getExperimentDetails, isNotFound } from 'services/api';
import { ExperimentDetailsParams } from 'services/types';
import { ExperimentDetails, RunState, TBSourceType } from 'types';
import { cancellableRunStates, killableRunStates, terminalRunStates } from 'utils/types';

import css from './ExperimentDetails.module.scss';
interface Params {
  experimentId: string;
}

const ExperimentDetailsComp: React.FC = () => {
  const { experimentId: experimentIdParam } = useParams<Params>();
  const experimentId = parseInt(experimentIdParam);

  interface ButtonLoadingStates {
    kill: boolean;
    [RunState.StoppingCanceled]: boolean;
    archive: boolean;
    [RunState.Paused]: boolean;
    [RunState.Active]: boolean;
    tsb: boolean; // tensorboard
  }

  const [ experiment, requestExperimentDetails ] =
  useRestApiSimple<ExperimentDetailsParams, ExperimentDetails>(
    getExperimentDetails, { id: experimentId });
  const pollExperimentDetails = useCallback(() => requestExperimentDetails({ id: experimentId }),
    [ requestExperimentDetails, experimentId ]);
  usePolling(pollExperimentDetails);

  const [ buttonStates, setButtonStates ] = useState<ButtonLoadingStates>({
    [RunState.StoppingCanceled]: false,
    archive: false,
    [RunState.Active]: false,
    kill: false,
    [RunState.Paused]: false,
    tsb: false,
  });

  const killExperimentCB = useCallback(() => {
    setButtonStates(state => ({ ...state, kill: true }));
    killExperiment({ experimentId })
      .then(pollExperimentDetails)
      .finally(() => setButtonStates(state => ({ ...state, kill: false })));
  }, [ experimentId, pollExperimentDetails ]);

  const archiveCB = useCallback((archive: boolean) =>
    (): Promise<unknown> => {
      setButtonStates(state => ({ ...state, archive: true }));
      return archiveExperiment(experimentId, archive)
        .then(pollExperimentDetails)
        .finally(() => setButtonStates(state => ({ ...state, archive: false })));
    },
  [ experimentId, pollExperimentDetails ]);

  const launchTensorboardCB = useCallback(() => {
    // TODO import from the tb PR.
    setButtonStates(state => ({ ...state, tsb: true }));
    launchTensorboard({ ids: [ experimentId ], type: TBSourceType.Experiment })
      .then(pollExperimentDetails)
      .finally(() => setButtonStates(state => ({ ...state, tsb: false })));
  }, [ experimentId, pollExperimentDetails ]);

  const requestExpStateCB = useCallback((targetState: RunState) =>
    (): Promise<unknown> => {
      setButtonStates(state => ({ ...state, [targetState]: true }));
      return setExperimentState({ experimentId, state: targetState })
        .then(pollExperimentDetails)
        .finally(() => setButtonStates(state => ({ ...state, [targetState]: false })));
    }
  , [ experimentId, pollExperimentDetails ]);

  if (isNaN(experimentId)) {
    return (
      <Page hideTitle title="Not Found">
        <Message>Bad experiment ID {experimentIdParam}</Message>
      </Page>
    );
  }

  if (experiment.error !== undefined) {
    const message = isNotFound(experiment.error) ? `Experiment ${experimentId} not found.`
      : `Failed to fetch experiment ${experimentId}.`;
    return (
      <Page hideTitle title="Not Found">
        <Message>{message}</Message>
      </Page>
    );
  }

  if (!experiment.data || experiment.isLoading) {
    return <Spinner fillContainer />;
  }

  const archiveButton = <Button key="archive" loading={buttonStates.archive}
    type="primary" onClick={archiveCB(true)}>
    Archive</Button>;
  const unarchiveButton = <Button key="unarchive" loading={buttonStates.archive}
    type="primary" onClick={archiveCB(false)}>
    Unarchive</Button>;

  const forkButton = <Button disabled key="fork" type="primary">Fork</Button>;
  const pauseButton = <Button key="pause" loading={buttonStates[RunState.Paused]}
    type="primary" onClick={requestExpStateCB(RunState.Paused)}>
    Pause</Button>;
  const activateButton = <Button key="activate" loading={buttonStates[RunState.Active]}
    type="primary" onClick={requestExpStateCB(RunState.Active)}>
    Activate</Button>;

  const cancelButton = <Popconfirm
    cancelText="No"
    key="cancel"
    okText="Yes"
    title="Are you sure you want to kill the experiment?"
    onConfirm={requestExpStateCB(RunState.StoppingCanceled)}
  >
    <Button danger loading={buttonStates[RunState.StoppingCanceled]}
      type="primary">
    Cancel</Button>
  </Popconfirm>;

  const killButton = <Popconfirm
    cancelText="No"
    key="kill"
    okText="Yes"
    title="Are you sure you want to kill the experiment?"
    onConfirm={killExperimentCB}
  >
    <Button danger loading={buttonStates.kill} type="primary">Kill</Button>
  </Popconfirm>;

  const tsbButton = <Button key="tensorboard"
    loading={buttonStates.tsb} type="primary" onClick={launchTensorboardCB}>
      Launch Tensorboard</Button>;

  interface ConditionalButton {
    btn: React.ReactNode;
    showIf?: (exp: ExperimentDetails) => boolean;
  }

  const actionButtons: ConditionalButton[] = [
    { btn: forkButton },
    {
      btn: archiveButton,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && !exp.archived,
    },
    {
      btn: tsbButton,
    },
    {
      btn: unarchiveButton,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && exp.archived,
    },
    {
      btn: pauseButton,
      showIf: (exp): boolean => exp.state === RunState.Active,
    },
    {
      btn: activateButton,
      showIf: (exp): boolean => exp.state === RunState.Paused,
    },
    {
      btn: cancelButton,
      showIf: (exp): boolean => !cancellableRunStates.includes(exp.state),
    },
    {
      btn: killButton,
      showIf: (exp): boolean => killableRunStates.includes(exp.state),
    },
  ];

  return (
    <Page className={css.base} title={`Experiment ${experiment.data?.config.description}`}>
      <Breadcrumb>
        <Breadcrumb.Item>
          <Space align="center" size="small">
            <Icon name="experiment" size="small" />
            <Link path="/det/experiments">Experiments</Link>
          </Space>
        </Breadcrumb.Item>
        <Breadcrumb.Item>
          <span>{experimentId}</span>
        </Breadcrumb.Item>
      </Breadcrumb>
      <ul>
        {actionButtons
          .filter(ab => !ab.showIf || ab.showIf(experiment.data as ExperimentDetails))
          .map(ab => ab.btn)
        }
      </ul>
      <ExperimentInfoBox experiment={experiment.data} />
      <Section title="Chart" />
      <Section title="Trials" />

    </Page>
  );
};

export default ExperimentDetailsComp;
