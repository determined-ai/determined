import { Alert, Breadcrumb, Button, Modal, Space, Table } from 'antd';
import { ColumnsType } from 'antd/lib/table';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';
import { useParams } from 'react-router';

import CheckpointModal from 'components/CheckpointModal';
import ExperimentActions from 'components/ExperimentActions';
import ExperimentInfoBox from 'components/ExperimentInfoBox';
import Icon from 'components/Icon';
import { makeClickHandler } from 'components/Link';
import Link from 'components/Link';
import Message from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import { stateRenderer } from 'components/Table';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { routeAll } from 'routes';
import { forkExperiment, getExperimentDetails, isNotFound } from 'services/api';
import { ExperimentDetailsParams } from 'services/types';
import { CheckpointDetail, ExperimentDetails, TrialSummary } from 'types';
import { clone } from 'utils/data';
import { alphanumericSorter, numericSorter, runStateSorter } from 'utils/data';
import { humanReadableFloat } from 'utils/string';

import css from './ExperimentDetails.module.scss';

interface Params {
  experimentId: string;
}

const ExperimentDetailsComp: React.FC = () => {
  const { experimentId } = useParams<Params>();
  const id = parseInt(experimentId);
  const [ experimentResponse, setExpRequestParams ] =
    useRestApiSimple<ExperimentDetailsParams, ExperimentDetails>(getExperimentDetails, { id });
  const experiment = experimentResponse.data;
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointDetail>();
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);

  const pollExperimentDetails = useCallback(() => {
    setExpRequestParams({ id });
  }, [ id, setExpRequestParams ]);

  usePolling(pollExperimentDetails);
  const [ forkValue, setForkValue ] = useState<string>('Loading');
  const [ forkModalState, setForkModalState ] = useState({ visible: false });
  const [ forkError, setForkError ] = useState<string>();

  useEffect(() => {
    if (experiment && experiment.config) {
      try {
        const prefix = 'Fork of ';
        const rawConfig = clone(experiment.configRaw);
        rawConfig.description = prefix + rawConfig.description;
        setForkValue(yaml.safeDump(rawConfig));
      } catch (e) {
        handleError({
          error: e,
          message: 'failed to load experiment config',
          type: ErrorType.ApiBadResponse,
        });
        setForkValue('failed to load experiment config');
      }
    }
  }, [ experiment ]);

  const showForkModal = useCallback((): void => {
    setForkModalState(state => ({ ...state, visible: true }));
  }, [ setForkModalState ]);

  const editorOnChange = useCallback((newValue) => {
    setForkValue(newValue);
    setForkError(undefined);
  }, []);

  const handleTableRow = useCallback((record: TrialSummary) => ({
    onClick: makeClickHandler(record.url as string),
  }), []);

  let message = '';
  if (isNaN(id)) message = `Bad experiment ID ${experimentId}`;
  if (experimentResponse.error) {
    message = isNotFound(experimentResponse.error) ?
      `Experiment ${experimentId} not found.` :
      `Failed to fetch experiment ${experimentId}.`;
  }
  if (message) {
    return (
      <Page hideTitle title="Not Found">
        <Message>{message}</Message>
      </Page>
    );
  } else if (!experiment) {
    return <Spinner fillContainer />;
  }

  const monacoOpts = {
    minimap: { enabled: false },
    selectOnLineNumbers: true,
  };

  const handleOk = async (): Promise<void> => {
    try {
      // Validate the yaml syntax by attempting to load it.
      yaml.safeLoad(forkValue);
      const forkId = await forkExperiment({ experimentConfig: forkValue, parentId: experimentId });
      setForkModalState(state => ({ ...state, visible: false }));
      routeAll(`/det/experiments/${forkId}`);
    } catch (e) {
      let errorMessage = 'Failed to fork using the provided config.';
      if (e.name === 'YAMLException') {
        errorMessage = e.message;
      } else if (e.response?.data?.message) {
        errorMessage = e.response.data.message;
      }
      setForkError(errorMessage);
    }
  };

  const handleCancel = (): void => {
    setForkModalState(state => ({ ...state, visible: false }));
  };

  const handleCheckpointShow = (event: React.MouseEvent, checkpoint: CheckpointDetail) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };
  const handleCheckpointDismiss = () => setShowCheckpoint(false);

  const columns: ColumnsType<TrialSummary> = [
    {
      dataIndex: 'id',
      sorter: (a: TrialSummary, b: TrialSummary): number => alphanumericSorter(a.id, b.id),
      title: 'ID',
    },
    {
      render: (_: string, record: TrialSummary): React.ReactNode => {
        if (experiment.config && record.bestAvailableCheckpoint) {
          const checkpoint: CheckpointDetail = {
            ...record.bestAvailableCheckpoint,
            batch: record.numBatchTally,
            experimentId: id,
            trialId: record.id,
          };
          return <Button onClick={e => handleCheckpointShow(e, checkpoint)}>
            {record.numBatchTally}
          </Button>;
        }
        return record.numBatchTally;
      },
      sorter: (a: TrialSummary, b: TrialSummary): number =>{
        return alphanumericSorter(a.numBatchTally, b.numBatchTally);
      },
      title: 'Batches',
    },
    {
      dataIndex: 'bestValidationMetric',
      render: (_: string, record: TrialSummary): React.ReactNode => {
        return record.bestValidationMetric ? humanReadableFloat(record.bestValidationMetric) : null;
      },
      sorter: (a: TrialSummary, b: TrialSummary): number => {
        return numericSorter(a.bestValidationMetric, b.bestValidationMetric);
      },
      title: `Metric (${experiment.config.searcher.metric})`,
    },
    {
      render: stateRenderer,
      sorter: (a: TrialSummary, b: TrialSummary): number => runStateSorter(a.state, b.state),
      title: 'State',
    },
  ];

  return (
    <Page title={`Experiment ${experiment?.config.description}`}>
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
      <ExperimentActions
        experiment={experiment}
        onClick={{ Fork: showForkModal }}
        onSettled={pollExperimentDetails} />
      <ExperimentInfoBox experiment={experiment} />
      <Modal
        bodyStyle={{
          padding: 0,
        }}
        className={css.forkModal}
        okText="Fork"
        style={{
          minWidth: '60rem',
        }}
        title={`Fork Experiment ${experimentId}`}
        visible={forkModalState.visible}
        onCancel={handleCancel}
        onOk={handleOk}
      >
        <MonacoEditor
          height="40vh"
          language="yaml"
          options={monacoOpts}
          theme="vs-light"
          value={forkValue}
          onChange={editorOnChange}
        />
        {forkError &&
          <Alert className={css.error} message={forkError} type="error" />
        }
      </Modal>
      <Section title="Chart" />
      <Section title="Trials">
        <Table
          columns={columns}
          dataSource={experiment?.trials}
          loading={!experimentResponse.hasLoaded}
          rowKey="id"
          size="small"
          onRow={handleTableRow} />
      </Section>
      {activeCheckpoint && <CheckpointModal
        checkpoint={activeCheckpoint}
        config={experiment.config}
        show={showCheckpoint}
        onHide={handleCheckpointDismiss} />}
    </Page>
  );
};

export default ExperimentDetailsComp;
