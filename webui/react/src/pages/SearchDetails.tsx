import Pivot, { PivotProps } from 'hew/Pivot';
import Notes from 'hew/RichTextEditor';
import { Loadable } from 'hew/utils/loadable';
import { string } from 'io-ts';
import _ from 'lodash';
import { useObservable } from 'micro-observables';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { unstable_useBlocker, useLocation, useNavigate, useParams } from 'react-router-dom';

import Page, { BreadCrumbRoute } from 'components/Page';
import { terminalRunStates } from 'constants/states';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getExperimentDetails, patchExperiment } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { ExperimentBase, Note, ValueOf, Workspace } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { isAborted, isNotFound } from 'utils/service';

import ExperimentCodeViewer from './ExperimentDetails/ExperimentCodeViewer';
import ExperimentDetailsHeader from './ExperimentDetails/ExperimentDetailsHeader';
import FlatRuns from './FlatRuns/FlatRuns';

const TabType = {
  Code: 'code',
  Notes: 'notes',
  Runs: 'runs',
} as const;

type TabType = ValueOf<typeof TabType>;

type Params = {
  searchId: string;
  tab?: TabType;
};
const TAB_KEYS = Object.values(TabType);
const INITIAL_TAB_KEY = TabType.Runs;

const SearchDetails: React.FC = () => {
  const { tab, searchId } = useParams<Params>();
  const [experiment, setExperiment] = useState<ExperimentBase>();
  const [pageError, setPageError] = useState<Error>();
  const canceler = useRef<AbortController>();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));
  const id = parseInt(searchId ?? '');

  const navigate = useNavigate();
  const location = useLocation();
  const [tabKey, setTabKey] = useState<TabType>(
    tab && TAB_KEYS.includes(tab) ? tab : INITIAL_TAB_KEY,
  );
  const basePath = paths.searchDetails(id);

  const configForExperiment = (experimentId: number): SettingsConfig<{ filePath: string }> => ({
    settings: {
      filePath: {
        defaultValue: '',
        storageKey: 'filePath',
        type: string,
      },
    },
    storagePath: `selected-file-${experimentId}`,
  });
  const config: SettingsConfig<{ filePath: string }> = useMemo(() => {
    return configForExperiment(id);
  }, [id]);
  const { settings, updateSettings } = useSettings<{ filePath: string }>(config);
  const handleSelectFile = useCallback(
    (filePath: string) => {
      updateSettings({ filePath });
    },
    [updateSettings],
  );

  const { canModifyExperimentMetadata, canViewExperimentArtifacts } = usePermissions();
  const showExperimentArtifacts =
    experiment &&
    canViewExperimentArtifacts({
      workspace: { id: experiment.workspaceId },
    });
  const editableNotes =
    experiment &&
    canModifyExperimentMetadata({
      workspace: { id: experiment.workspaceId },
    });

  const handleTabChange = useCallback(
    (key: string) => {
      navigate(`${basePath}/${key}`, { replace: true });
    },
    [basePath, navigate],
  );

  useEffect(() => {
    setTabKey(tab ?? INITIAL_TAB_KEY);
  }, [location.pathname, tab]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      if (window.location.pathname.includes(basePath))
        navigate(`${basePath}/${tabKey}`, { replace: true });
    }
  }, [basePath, navigate, tab, tabKey]);

  const fetchExperimentDetails = useCallback(async () => {
    if (!searchId) return;
    try {
      const newExperiment = await getExperimentDetails(
        { id: parseInt(searchId) },
        { signal: canceler.current?.signal },
      );
      setExperiment((prevExperiment) =>
        _.isEqual(prevExperiment, newExperiment) ? prevExperiment : newExperiment,
      );
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [pageError, searchId]);

  const handleNotesUpdate = useCallback(
    async (notes: Note) => {
      const editedNotes = notes.contents;
      try {
        await patchExperiment({ body: { notes: editedNotes }, experimentId: id });
        await fetchExperimentDetails();
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to update search notes.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    },
    [id, fetchExperimentDetails],
  );

  const tabItems: PivotProps['items'] = useMemo(() => {
    if (!experiment) return [];

    const tabItems: PivotProps['items'] = [
      {
        children: experiment?.projectId && (
          <FlatRuns
            projectId={experiment.projectId}
            searchId={id}
            workspaceId={experiment.workspaceId}
          />
        ),
        key: TabType.Runs,
        label: 'Runs',
      },
    ];

    if (showExperimentArtifacts && experiment.modelDefinitionSize !== 0) {
      tabItems.push({
        children: experiment && showExperimentArtifacts && (
          <ExperimentCodeViewer
            experiment={experiment}
            selectedFilePath={settings.filePath}
            onSelectFile={handleSelectFile}
          />
        ),
        key: TabType.Code,
        label: 'Code',
      });
    }

    tabItems.push({
      children: (
        <Notes
          disabled={!editableNotes}
          disableTitle
          docs={{ contents: experiment?.notes ?? '', name: 'Notes' }}
          onError={handleError}
          onPageUnloadHook={unstable_useBlocker}
          onSave={handleNotesUpdate}
        />
      ),
      key: TabType.Notes,
      label: 'Notes',
    });

    return tabItems;
  }, [
    editableNotes,
    experiment,
    handleSelectFile,
    handleNotesUpdate,
    id,
    settings.filePath,
    showExperimentArtifacts,
  ]);

  const { stopPolling } = usePolling(fetchExperimentDetails, { rerunOnNewFn: true });

  useEffect(() => {
    if (experiment && terminalRunStates.has(experiment.state)) {
      stopPolling();
    }
  }, [experiment, stopPolling]);

  const workspaceName = workspaces.find((ws: Workspace) => ws.id === experiment?.workspaceId)?.name;

  const pageBreadcrumb: BreadCrumbRoute[] = [
    workspaceName && experiment?.workspaceId !== 1
      ? {
          breadcrumbName: workspaceName,
          path: paths.workspaceDetails(experiment?.workspaceId ?? 1),
        }
      : {
          breadcrumbName: 'Uncategorized Runs',
          path: paths.projectDetails(1),
        },
  ];

  if (experiment?.projectName && experiment?.projectId && experiment?.projectId !== 1)
    pageBreadcrumb.push({
      breadcrumbName: experiment?.projectName ?? '',
      path: paths.projectDetails(experiment?.projectId),
    });

  pageBreadcrumb.push({
    breadcrumbName: experiment?.name ?? '',
    path: paths.searchDetails(id),
  });

  return (
    <Page
      breadcrumb={pageBreadcrumb}
      headerComponent={
        experiment && (
          <ExperimentDetailsHeader
            experiment={experiment}
            fetchExperimentDetails={fetchExperimentDetails}
          />
        )
      }
      notFound={pageError && isNotFound(pageError)}
      stickyHeader
      title={`Search ${searchId}`}>
      <Pivot activeKey={tabKey} items={tabItems} onChange={handleTabChange} />
    </Page>
  );
};

export default SearchDetails;
