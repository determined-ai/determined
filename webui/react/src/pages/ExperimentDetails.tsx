import { Button, Col, Row, Space, Tooltip } from 'antd';
import { SorterResult } from 'antd/es/table/interface';
import axios from 'axios';
import yaml from 'js-yaml';
import React, { ReactNode, useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router';

import Badge, { BadgeType } from 'components/Badge';
import CheckpointModal from 'components/CheckpointModal';
import CreateExperimentModal from 'components/CreateExperimentModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Icon from 'components/Icon';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import Spinner, { Indicator } from 'components/Spinner';
import { defaultRowClassName, getPaginationConfig, humanReadableFloatRenderer,
  MINIMUM_PAGE_SIZE } from 'components/Table';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import ExperimentActions from 'pages/ExperimentDetails/ExperimentActions';
import ExperimentChart from 'pages/ExperimentDetails/ExperimentChart';
import ExperimentInfoBox, { TopWorkloads } from 'pages/ExperimentDetails/ExperimentInfoBox';
import { handlePath, paths } from 'routes/utils';
import { getExperimentDetails, getExpTrials, getExpValidationHistory,
  isNotFound } from 'services/api';
import { detApi } from 'services/apiConfig';
import { decodeCheckpoint } from 'services/decoder';
import { ApiSorter, ApiState } from 'services/types';
import { isAborted } from 'services/utils';
import { CheckpointWorkloadExtended, ExperimentBase, TrialItem,
  ValidationHistory } from 'types';
import { clone, numericSorter } from 'utils/data';
import { getMetricValue, terminalRunStates, upgradeConfig } from 'utils/types';

import css from './ExperimentDetails.module.scss';
import { columns as defaultColumns } from './ExperimentDetails/ExperimentOverview.table';

interface Params {
  experimentId: string;
}

const STORAGE_PATH = 'experiment-detail';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';

const ExperimentDetailsComp: React.FC = () => {
  const { experimentId } = useParams<Params>();
  const id = parseInt(experimentId);
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointWorkloadExtended>();
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const [ forkModalVisible, setForkModalVisible ] = useState(false);
  const [ forkModalConfig, setForkModalConfig ] = useState('Loading');
  const storage = useStorage(STORAGE_PATH);
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initSorter: ApiSorter | null = storage.get(STORAGE_SORTER_KEY);
  const [ pageSize, setPageSize ] = useState(initLimit);
  const [ sorter, setSorter ] = useState<ApiSorter | null>(initSorter);
  const [ experimentDetails, setExperimentDetails ] = useState<ApiState<ExperimentBase>>({
    data: undefined,
    error: undefined,
    isLoading: true,
    source: axios.CancelToken.source(),
  });
  const [ experimentCanceler ] = useState(new AbortController());
  const [ trials, setTrials ] = useState<TrialItem[]>([]);
  const [ valHistory, setValHistory ] = useState<ValidationHistory[]>([]);
  const [ bestWorkloads, setBestWorkloads ] = useState<TopWorkloads>();

  const experiment = experimentDetails.data;
  const experimentConfig = experiment?.config;

  useEffect(() => {
    if (id === undefined) return;
    (async () => {
      const resp = await detApi.Experiments.determinedGetExperimentCheckpoints(
        id,
        'SORT_BY_SEARCHER_METRIC',
        experimentConfig?.searcher.smallerIsBetter ? 'ORDER_BY_ASC' : 'ORDER_BY_DESC',
        undefined,
        1,
        [ 'STATE_COMPLETED' ],
        [ 'STATE_COMPLETED' ],
      );
      const checkpoints = resp.checkpoints?.map(decodeCheckpoint);
      const bestCheckpoint = checkpoints && checkpoints[0];

      const bestValidation = valHistory.length > 1 ?
        valHistory[valHistory.length-1]?.validationError : undefined;

      setBestWorkloads({ bestCheckpoint, bestValidation });
    })();
  }, [ id, valHistory, experimentConfig?.searcher.smallerIsBetter ]);

  const columns = useMemo(() => {
    const latestValidationRenderer = (_: string, record: TrialItem): React.ReactNode => {
      const value = getMetricValue(record.latestValidationMetric, metric);
      return value && <HumanReadableFloat num={value} />;
    };

    const latestValidationSorter = (a: TrialItem, b: TrialItem): number => {
      if (!metric) return 0;
      const aMetric = getMetricValue(a.latestValidationMetric, metric);
      const bMetric = getMetricValue(b.latestValidationMetric, metric);
      return numericSorter(aMetric, bMetric);
    };

    const checkpointRenderer = (_: string, record: TrialItem): React.ReactNode => {
      if (!record.bestAvailableCheckpoint) return;
      const checkpoint: CheckpointWorkloadExtended = {
        ...record.bestAvailableCheckpoint,
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
      if (column.key === 'bestValidation') {
        column.render = (_: string, record: TrialItem): ReactNode => {
          const value = getMetricValue(record.bestValidationMetric, metric);
          return value && humanReadableFloatRenderer(value);
        };
      }
      return column;
    });

    return newColumns;
  }, [ experimentConfig, id, sorter ]);

  const fetchExperimentDetails = useCallback(async () => {
    try {
      const experiment = await getExperimentDetails({ id, signal: experimentCanceler.signal });
      const trials = await getExpTrials({ id });
      const validationHistory = await getExpValidationHistory({ id });
      setExperimentDetails(prev => ({ ...prev, data: experiment, isLoading: false }));
      setTrials(trials);
      setValHistory(validationHistory);
    } catch (e) {
      if (!experimentDetails.error && !isAborted(e)) {
        setExperimentDetails(prev => ({ ...prev, error: e }));
      }
    }
  }, [ id, experimentDetails.error, experimentCanceler.signal ]);

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

  const handleTableChange = useCallback((tablePagination, tableFilters, sorter) => {
    if (Array.isArray(sorter)) return;

    const { columnKey, order } = sorter as SorterResult<TrialItem>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    storage.set(STORAGE_SORTER_KEY, { descend: order === 'descend', key: columnKey as string });
    setSorter({ descend: order === 'descend', key: columnKey as string });

    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPageSize(tablePagination.pageSize);
  }, [ columns, setSorter, storage ]);

  const handleTableRow = useCallback((record: TrialItem) => {
    const handleClick = (event: React.MouseEvent) =>
      handlePath(event, { path: paths.trialDetails(record.id, id) });
    return { onAuxClick: handleClick, onClick: handleClick };
  }, [ id ]);

  const handleCheckpointShow = (
    event: React.MouseEvent,
    checkpoint: CheckpointWorkloadExtended,
  ) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };

  const handleCheckpointDismiss = useCallback(() => setShowCheckpoint(false), []);

  const stopPolling = usePolling(fetchExperimentDetails);
  useEffect(() => {
    if (experimentDetails.data && terminalRunStates.has(experimentDetails.data.state)) {
      stopPolling();
    }
  }, [ experimentDetails.data, stopPolling ]);

  useEffect(() => {
    return () => experimentDetails.source?.cancel();
  }, [ experimentDetails.source ]);

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
  if (experimentDetails.error) {
    const message = isNotFound(experimentDetails.error) ?
      `Unable to find Experiment ${experimentId}` :
      `Unable to fetch Experiment ${experimentId}`;
    return <Message title={message} type={MessageType.Warning} />;
  }
  if (!experiment) return <Spinner />;

  return (
    <Page
      breadcrumb={[
        { breadcrumbName: 'Experiments', path: '/experiments' },
        {
          breadcrumbName: `Experiment ${experimentId}`,
          path: `/experiments/${experimentId}`,
        },
      ]}
      options={<ExperimentActions
        experiment={experiment}
        trials={trials}
        onClick={{ Fork: showForkModal }}
        onSettled={fetchExperimentDetails} />}
      showDivider
      subTitle={<Space align="center" size="small">
        {experiment?.config.description}
        <Badge state={experiment.state} type={BadgeType.State} />
        {experiment.archived && <Badge>ARCHIVED</Badge>}
      </Space>}
      title={`Experiment ${experimentId}`}>
      <div className={css.base}>
        <Row className={css.topRow} gutter={[ 16, 16 ]}>
          <Col lg={10} span={24} xl={8} xxl={6}>
            <ExperimentInfoBox
              bestCheckpoint={bestWorkloads?.bestCheckpoint}
              bestValidation={bestWorkloads?.bestValidation}
              experiment={experiment}
              onTagsChange={fetchExperimentDetails}
            />
          </Col>
          <Col lg={14} span={24} xl={16} xxl={18}>
            <ExperimentChart
              startTime={experiment.startTime}
              validationHistory={valHistory}
              validationMetric={experimentConfig?.searcher.metric} />
          </Col>
          <Col span={24}>
            <Section title="Trials">
              <ResponsiveTable<TrialItem>
                columns={columns}
                dataSource={trials}
                loading={{
                  indicator: <Indicator />,
                  spinning: experimentDetails.isLoading,
                }}
                pagination={getPaginationConfig(trials.length || 0, pageSize)}
                rowClassName={defaultRowClassName({ clickable: true })}
                rowKey="id"
                showSorterTooltip={false}
                size="small"
                onChange={handleTableChange}
                onRow={handleTableRow} />
            </Section>
          </Col>
        </Row>
      </div>
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
