import { Alert, Breadcrumb, Button, Modal, Space, Table, Tooltip } from 'antd';
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
import { durationRenderer, relativeTimeRenderer, stateRenderer } from 'components/Table';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { routeAll } from 'routes';
import { forkExperiment, getExperimentDetails, isNotFound } from 'services/api';
import { ExperimentDetailsParams } from 'services/types';
import { CheckpointDetail, ExperimentDetails, TrialItem } from 'types';
import { clone } from 'utils/data';
import { alphanumericSorter, numericSorter, runStateSorter, stringTimeSorter } from 'utils/data';
import { humanReadableFloat } from 'utils/string';
import { getDuration } from 'utils/time';

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
  const validationKey = experiment?.config.searcher.metric;
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

  const handleTableRow = useCallback((record: TrialItem) => ({
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
      const forkId = await forkExperiment({ experimentConfig: forkValue, parentId: id });
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

  const columns: ColumnsType<TrialItem> = [
    {
      dataIndex: 'id',
      sorter: (a: TrialItem, b: TrialItem): number => alphanumericSorter(a.id, b.id),
      title: 'ID',
    },
    {
      render: stateRenderer,
      sorter: (a: TrialItem, b: TrialItem): number => runStateSorter(a.state, b.state),
      title: 'State',
    },
    {
      dataIndex: 'numBatches',
      sorter: (a: TrialItem, b: TrialItem): number => {
        return numericSorter(a.numBatches, b.numBatches);
      },
      title: 'Batches',
    },
    {
      defaultSortOrder: experiment.config.searcher.smallerIsBetter ? 'ascend' : 'descend',
      render: (_: string, record: TrialItem): React.ReactNode => {
        return record.bestValidationMetric ? humanReadableFloat(record.bestValidationMetric) : '-';
      },
      sorter: (a: TrialItem, b: TrialItem): number => {
        return numericSorter(a.bestValidationMetric, b.bestValidationMetric);
      },
      title: 'Best Validation Metric',
    },
    {
      render: (_: string, record: TrialItem): React.ReactNode => {
        return record.latestValidationMetrics && validationKey ?
          humanReadableFloat(record.latestValidationMetrics.validationMetrics[validationKey]) :
          '-';
      },
      sorter: (a: TrialItem, b: TrialItem): number => {
        if (!validationKey) return 0;
        const aMetric = a.latestValidationMetrics?.validationMetrics[validationKey];
        const bMetric = b.latestValidationMetrics?.validationMetrics[validationKey];
        return numericSorter(aMetric, bMetric);
      },
      title: 'Latest Validation Metric',
    },
    {
      render: (_: string, record: TrialItem): React.ReactNode => {
        return relativeTimeRenderer(new Date(record.startTime));
      },
      sorter: (a: TrialItem, b: TrialItem): number => {
        return stringTimeSorter(a.startTime, b.startTime);
      },
      title: 'Start Time',
    },
    {
      render: (_: string, record: TrialItem): React.ReactNode => durationRenderer(record),
      sorter: (a: TrialItem, b: TrialItem): number => getDuration(a) - getDuration(b),
      title: 'Duration',
    },
    {
      render: (_: string, record: TrialItem): React.ReactNode => {
        if (record.bestAvailableCheckpoint) {
          const checkpoint: CheckpointDetail = {
            ...record.bestAvailableCheckpoint,
            batch: record.numBatches,
            experimentId: id,
            trialId: record.id,
          };
          return <Tooltip title="View Checkpoint">
            <Button
              aria-label="View Checkpoint"
              icon={<Icon name="checkpoint" />}
              onClick={e => handleCheckpointShow(e, checkpoint)} />
          </Tooltip>;
        }
        return '-';
      },
      title: 'Checkpoint',
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
        title={`Best Checkpoint for Trial ${activeCheckpoint.trialId}`}
        onHide={handleCheckpointDismiss} />}
    </Page>
  );
};

export default ExperimentDetailsComp;
