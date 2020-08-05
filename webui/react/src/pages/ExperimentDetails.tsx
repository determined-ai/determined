import { Button, Col, Row, Space, Table, Tooltip } from 'antd';
import { ColumnType } from 'antd/lib/table';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router';

import Badge, { BadgeType } from 'components/Badge';
import CheckpointModal from 'components/CheckpointModal';
import CreateExperimentModal from 'components/CreateExperimentModal';
import Icon from 'components/Icon';
import { makeClickHandler } from 'components/Link';
import Message from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import { durationRenderer, relativeTimeRenderer, stateRenderer } from 'components/Table';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import ExperimentActions from 'pages/ExperimentDetails/ExperimentActions';
import ExperimentChart from 'pages/ExperimentDetails/ExperimentChart';
import ExperimentInfoBox from 'pages/ExperimentDetails/ExperimentInfoBox';
import { getExperimentDetails, isNotFound } from 'services/api';
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
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointDetail>();
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const [ experimentResponse, triggerExperimentRequest ] =
    useRestApi<ExperimentDetailsParams, ExperimentDetails>(getExperimentDetails, { id });
  const experiment = experimentResponse.data;
  const experimentConfig = experiment?.config;
  const validationKey = experiment?.config.searcher.metric;

  const pollExperimentDetails = useCallback(() => {
    triggerExperimentRequest({ id });
  }, [ id, triggerExperimentRequest ]);

  usePolling(pollExperimentDetails);
  const [ forkModalVisible, setForkModalVisible ] = useState(false);
  const [ forkModalConfig, setForkModalConfig ] = useState('Loading');

  useEffect(() => {
    if (experiment && experiment.config) {
      try {
        const prefix = 'Fork of ';
        const rawConfig = clone(experiment.configRaw);
        rawConfig.description = prefix + rawConfig.description;
        setForkModalConfig(yaml.safeDump(rawConfig));
      } catch (e) {
        handleError({
          error: e,
          message: 'failed to load experiment config',
          type: ErrorType.ApiBadResponse,
        });
        setForkModalConfig('failed to load experiment config');
      }
    }
  }, [ experiment ]);

  const showForkModal = useCallback((): void => {
    setForkModalVisible(true);
  }, [ setForkModalVisible ]);

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
      <Page id="page-error-message">
        <Message>{message}</Message>
      </Page>
    );
  } else if (!experiment) {
    return <Spinner fillContainer />;
  }

  const handleCheckpointShow = (event: React.MouseEvent, checkpoint: CheckpointDetail) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };
  const handleCheckpointDismiss = () => setShowCheckpoint(false);

  const columns: ColumnType<TrialItem>[] = [
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
    <Page
      backPath={'/det/experiments'}
      breadcrumb={[
        { breadcrumbName: 'Experiments', path: '/det/experiments' },
        { breadcrumbName: `Experiment ${experimentId}`, path: `/det/experiments/${experimentId}` },
      ]}
      options={<ExperimentActions
        experiment={experiment}
        onClick={{ Fork: showForkModal }}
        onSettled={pollExperimentDetails} />}
      subTitle={<Space align="center" size="small">
        {experiment?.config.description}
        <Badge state={experiment.state} type={BadgeType.State} />
      </Space>}
      title={`Experiment ${experimentId}`}>
      <Row className={css.topRow} gutter={[ 16, 16 ]}>
        <Col lg={10} span={24} xl={8} xxl={6}>
          <ExperimentInfoBox experiment={experiment} />
        </Col>
        <Col lg={14} span={24} xl={16} xxl={18}>
          <ExperimentChart
            startTime={experiment.startTime}
            validationHistory={experiment.validationHistory}
            validationMetric={experimentConfig?.searcher.metric} />
        </Col>
        <Col span={24}>
          <Section title="Trials">
            <Table
              columns={columns}
              dataSource={experiment?.trials}
              loading={!experimentResponse.hasLoaded}
              rowKey="id"
              size="small"
              onRow={handleTableRow} />
          </Section>
        </Col>
      </Row>
      {activeCheckpoint && <CheckpointModal
        checkpoint={activeCheckpoint}
        config={experiment.config}
        show={showCheckpoint}
        title={`Best Checkpoint for Trial ${activeCheckpoint.trialId}`}
        onHide={handleCheckpointDismiss} />}
      <CreateExperimentModal
        config={forkModalConfig}
        okText="Fork"
        parentId={id}
        title={`Fork Experiment ${id}`}
        visible={forkModalVisible}
        onConfigChange={setForkModalConfig}
        onVisibleChange={setForkModalVisible}
      />
    </Page>
  );
};

export default ExperimentDetailsComp;
