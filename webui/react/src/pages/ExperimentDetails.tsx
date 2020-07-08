import { Breadcrumb, Button, Popconfirm, Space } from 'antd';
import React, { useCallback } from 'react';
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
import { killExperiment, launchTensorboard, setExperimentState } from 'services/api';
import { getExperimentDetails, isNotFound } from 'services/api';
import { ExperimentDetailsParams } from 'services/types';
import { ExperimentDetails, RunState, TBSourceType } from 'types';
import { cancellableRunStates, killableRunStates } from 'utils/types';

import css from './ExperimentDetails.module.scss';
interface Params {
  experimentId: string;
}

const ExperimentDetailsComp: React.FC = () => {
  const { experimentId: experimentIdParam } = useParams<Params>();
  const experimentId = parseInt(experimentIdParam);

  const [ experiment, requestExperimentDetails ] =
  useRestApiSimple<ExperimentDetailsParams, ExperimentDetails>(
    getExperimentDetails, { id: experimentId });
  usePolling(() => requestExperimentDetails);

  const killExperimentCB = useCallback(() => {
    killExperiment({ experimentId });
  }, [ experimentId ]);

  const launchTensorboardCB = useCallback(() => {
    // TODO import from the tb PR.
    launchTensorboard({ ids: [ experimentId ], type: TBSourceType.Experiment });
  }, [ experimentId ]);

  const requestExpStateCB = useCallback((state: RunState) => {
    return (): Promise<void> => setExperimentState({ experimentId, state });
  }, [ experimentId ]);

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

  const forkButton = <Button disabled key="fork" type="primary">Fork</Button>;
  const pauseButton = <Button key="pause" type="primary"
    onClick={requestExpStateCB(RunState.Paused)}>
    Pause</Button>;
  const activateButton = <Button key="activate" type="primary"
    onClick={requestExpStateCB(RunState.Active)}>
    Activate</Button>;

  const cancelButton = <Popconfirm
    cancelText="No"
    okText="Yes"
    title="Are you sure you want to kill the experiment?"
    onConfirm={killExperimentCB}
  >
    <Button danger key="cancel" type="primary"
      onClick={requestExpStateCB(RunState.StoppingCanceled)}>
    Cancel</Button>
  </Popconfirm>;

  const killButton = <Popconfirm
    cancelText="No"
    okText="Yes"
    title="Are you sure you want to kill the experiment?"
    onConfirm={killExperimentCB}
  >
    <Button danger key="kill" type="primary">Kill</Button>
  </Popconfirm>;

  const tsbButton = <Button key="tensorboard"
    type="primary" onClick={launchTensorboardCB}> Launch Tensorboard</Button>;

  interface ConditionalButton {
    btn: React.ReactNode;
    showIf?: (exp: ExperimentDetails) => boolean;
  }

  const actionButtons: ConditionalButton[] = [
    { btn: forkButton },
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
      showIf: (exp): boolean => cancellableRunStates.includes(exp.state),
    },
    {
      btn: killButton,
      showIf: (exp): boolean => killableRunStates.includes(exp.state),
    },
    {
      btn: tsbButton,
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
