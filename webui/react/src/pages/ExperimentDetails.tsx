import { Button, Col, Row, Space, Table, Tooltip } from 'antd';
import { SorterResult } from 'antd/es/table/interface';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router';

import Badge, { BadgeType } from 'components/Badge';
import CheckpointModal from 'components/CheckpointModal';
import CreateExperimentModal from 'components/CreateExperimentModal';
import Icon from 'components/Icon';
import Message from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner, { Indicator } from 'components/Spinner';
import { defaultRowClassName, getPaginationConfig } from 'components/Table';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useRestApi from 'hooks/useRestApi';
import useStorage from 'hooks/useStorage';
import ExperimentActions from 'pages/ExperimentDetails/ExperimentActions';
import ExperimentChart from 'pages/ExperimentDetails/ExperimentChart';
import ExperimentInfoBox from 'pages/ExperimentDetails/ExperimentInfoBox';
import { getExperimentDetails, isNotFound } from 'services/api';
import { ApiSorter, ExperimentDetailsParams } from 'services/types';
import { CheckpointDetail, ExperimentDetails, TrialItem } from 'types';
import { clone } from 'utils/data';
import { numericSorter } from 'utils/data';
import { handlePath } from 'utils/routes';
import { humanReadableFloat } from 'utils/string';
import { upgradeConfig } from 'utils/types';

import css from './ExperimentDetails.module.scss';
import { columns as defaultColumns } from './ExperimentDetails.table';

interface Params {
  experimentId: string;
}

const STORAGE_PATH = 'experiment-detail';
const STORAGE_SORTER_KEY = 'sorter';

const ExperimentDetailsComp: React.FC = () => {
  const { experimentId } = useParams<Params>();
  const id = parseInt(experimentId);
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointDetail>();
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const [ forkModalVisible, setForkModalVisible ] = useState(false);
  const [ forkModalConfig, setForkModalConfig ] = useState('Loading');
  const [ experimentResponse, triggerExperimentRequest ] =
    useRestApi<ExperimentDetailsParams, ExperimentDetails>(getExperimentDetails, { id });
  const storage = useStorage(STORAGE_PATH);
  const initSorter: ApiSorter | null = storage.get(STORAGE_SORTER_KEY);
  const [ sorter, setSorter ] = useState<ApiSorter | null>(initSorter);

  const experiment = experimentResponse.data;
  const experimentConfig = experiment?.config;

  const columns = useMemo(() => {
    const latestValidationRenderer = (_: string, record: TrialItem): React.ReactNode => {
      return record.latestValidationMetrics && metric &&
        humanReadableFloat(record.latestValidationMetrics.validationMetrics[metric]);
    };

    const latestValidationSorter = (a: TrialItem, b: TrialItem): number => {
      if (!metric) return 0;
      const aMetric = a.latestValidationMetrics?.validationMetrics[metric];
      const bMetric = b.latestValidationMetrics?.validationMetrics[metric];
      return numericSorter(aMetric, bMetric);
    };

    const checkpointRenderer = (_: string, record: TrialItem): React.ReactNode => {
      if (!record.bestAvailableCheckpoint) return;
      const checkpoint: CheckpointDetail = {
        ...record.bestAvailableCheckpoint,
        batch: record.totalBatchesProcessed,
        experimentId: id,
        trialId: record.id,
      };
      return (
        <Tooltip title="View Checkpoint">
          <Button
            aria-label="View Checkpoint"
            icon={<Icon name="checkpoint" />}
            onClick={e => handleCheckpointShow(e, checkpoint)} />
        </Tooltip>
      );
    };

    const { metric, smallerIsBetter } = experimentConfig?.searcher || {};
    const newColumns = [ ...defaultColumns ].map(column => {
      column.sortOrder = null;
      if (!sorter && column.key === 'bestValidation') {
        column.sortOrder = smallerIsBetter ? 'ascend' : 'descend';
      } else if (sorter && column.key === sorter.key) {
        column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      }
      if (column.key === 'latestValidation') {
        column.render = latestValidationRenderer;
        column.sorter = latestValidationSorter;
      }
      if (column.key === 'checkpoint') column.render = checkpointRenderer;
      return column;
    });

    return newColumns;
  }, [ experimentConfig, id, sorter ]);

  const pollExperimentDetails = useCallback(() => {
    triggerExperimentRequest({ id });
  }, [ id, triggerExperimentRequest ]);

  const setFreshForkConfig = useCallback(() => {
    if (!experiment?.configRaw) return;
    // do not reset the config if the modal is open
    if (forkModalVisible) return;
    const prefix = 'Fork of ';
    const rawConfig = clone(experiment.configRaw);
    rawConfig.description = prefix + rawConfig.description;
    upgradeConfig(rawConfig);
    setForkModalConfig(yaml.safeDump(rawConfig));
  }, [ experiment?.configRaw, forkModalVisible ]);

  const handleForkModalCancel = useCallback(() => {
    setForkModalVisible(false);
    setFreshForkConfig();
  }, [ setFreshForkConfig ]);

  const showForkModal = useCallback((): void => {
    setForkModalVisible(true);
  }, [ setForkModalVisible ]);

  const handleTableChange = useCallback((pagination, filters, sorter) => {
    if (Array.isArray(sorter)) return;

    const { columnKey, order } = sorter as SorterResult<TrialItem>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    storage.set(STORAGE_SORTER_KEY, { descend: order === 'descend', key: columnKey as string });
    setSorter({ descend: order === 'descend', key: columnKey as string });
  }, [ columns, setSorter, storage ]);

  const handleTableRow = useCallback((record: TrialItem) => {
    const handleClick = (event: React.MouseEvent) => handlePath(event, { path: record.url });
    return { onAuxClick: handleClick, onClick: handleClick };
  }, []);

  const handleCheckpointShow = (event: React.MouseEvent, checkpoint: CheckpointDetail) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };

  const handleCheckpointDismiss = useCallback(() => setShowCheckpoint(false), []);

  usePolling(pollExperimentDetails);

  useEffect(() => {
    try {
      setFreshForkConfig();
    } catch (e) {
      handleError({
        error: e,
        message: 'failed to load experiment config',
        type: ErrorType.ApiBadResponse,
      });
      setForkModalConfig('failed to load experiment config');
    }
  }, [ setFreshForkConfig ]);

  if (isNaN(id)) return <Message title={`Invalid Experiment ID ${experimentId}`} />;
  if (experimentResponse.error) {
    const message = isNotFound(experimentResponse.error) ?
      `Unable to find Experiment ${experimentId}` :
      `Unable to fetch Experiment ${experimentId}`;
    return <Message title={message} />;
  }
  if (!experiment) return <Spinner />;

  return (
    <Page
      backPath={'/det/experiments'}
      breadcrumb={[
        { breadcrumbName: 'Experiments', path: '/det/experiments' },
        {
          breadcrumbName: `Experiment ${experimentId}`,
          path: `/det/experiments/${experimentId}`,
        },
      ]}
      options={<ExperimentActions
        experiment={experiment}
        onClick={{ Fork: showForkModal }}
        onSettled={pollExperimentDetails} />}
      showDivider
      subTitle={<Space align="center" size="small">
        {experiment?.config.description}
        <Badge state={experiment.state} type={BadgeType.State} />
        {experiment.archived && <Badge>ARCHIVED</Badge>}
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
              loading={{
                indicator: <Indicator />,
                spinning: !experimentResponse.hasLoaded,
              }}
              pagination={getPaginationConfig(experiment?.trials.length || 0)}
              rowClassName={defaultRowClassName()}
              rowKey="id"
              showSorterTooltip={false}
              size="small"
              onChange={handleTableChange}
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
        onCancel={handleForkModalCancel}
        onConfigChange={setForkModalConfig}
        onVisibleChange={setForkModalVisible} />
    </Page>
  );
};

export default ExperimentDetailsComp;
