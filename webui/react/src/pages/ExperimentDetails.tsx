import { Alert, Breadcrumb, Modal, Space } from 'antd';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useState } from 'react';
import MonacoEditor from 'react-monaco-editor';
import { useParams } from 'react-router';

import ExperimentActions from 'components/ExperimentActions';
import ExperimentInfoBox from 'components/ExperimentInfoBox';
import Icon from 'components/Icon';
import Link from 'components/Link';
import Message from 'components/Message';
import Page from 'components/Page';
import Section from 'components/Section';
import Spinner from 'components/Spinner';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import { useRestApiSimple } from 'hooks/useRestApi';
import { routeAll } from 'routes';
import { forkExperiment, getExperimentDetails, isNotFound } from 'services/api';
import { ExperimentDetailsParams } from 'services/types';
import { ExperimentDetails } from 'types';
import { clone } from 'utils/data';

import css from './ExperimentDetails.module.scss';

interface Params {
  experimentId: string;
}

const ExperimentDetailsComp: React.FC = () => {
  const { experimentId: experimentIdParam } = useParams<Params>();
  const experimentId = parseInt(experimentIdParam);
  const [ experiment, setExpRequestParams ] =
  useRestApiSimple<ExperimentDetailsParams, ExperimentDetails>(
    getExperimentDetails, { id: experimentId });
  const pollExperimentDetails = useCallback(() => setExpRequestParams({ id: experimentId }),
    [ setExpRequestParams, experimentId ]);
  usePolling(pollExperimentDetails);
  const [ forkValue, setForkValue ] = useState<string>('Loading');
  const [ forkModalState, setForkModalState ] = useState({ visible: false });
  const [ forkError, setForkError ] = useState<string>();

  useEffect(() => {
    if (experiment.data && experiment.data.config) {
      try {
        const prefix = 'Fork of ';
        const rawConfig = clone(experiment.data.configRaw);
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
  }, [ experiment.data ]);

  const showForkModal = useCallback((): void => {
    setForkModalState(state => ({ ...state, visible: true }));
  }, [ setForkModalState ]);

  const editorOnChange = useCallback((newValue) => {
    setForkValue(newValue);
    setForkError(undefined);
  }, []);

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
  } else if (!experiment.data) {
    return <Spinner fillContainer />;
  }

  const monacoOpts = {
    minimap: {
      enabled: false,
    },
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

  return (
    <Page title={`Experiment ${experiment.data?.config.description}`}>
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
      <ExperimentActions experiment={experiment.data} onClick={{ Fork: showForkModal }}
        onSettled={pollExperimentDetails} />
      <ExperimentInfoBox experiment={experiment.data} />
      <Section title="Chart" />
      <Section title="Trials" />

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
    </Page>
  );
};

export default ExperimentDetailsComp;
