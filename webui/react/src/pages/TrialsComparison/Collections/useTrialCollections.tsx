import { Select } from 'antd';
import { string } from 'io-ts';
import { useCallback, useEffect, useMemo, useState } from 'react';

import Button from 'components/kit/Button';
import Dropdown from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { SettingsConfig, useSettings, UseSettingsReturn } from 'hooks/useSettings';
import useStorage from 'hooks/useStorage';
import { deleteTrialsCollection, getTrialsCollections, patchTrialsCollection } from 'services/api';
import userStore from 'stores/users';
import { clone, finiteElseUndefined, isFiniteNumber } from 'utils/data';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import { decodeTrialsCollection, encodeTrialsCollection } from '../api';

import { TrialsCollection } from './collections';
import { FilterSetter, SetFilters, TrialFilters, TrialSorter } from './filters';
import useModalTrialCollection, { CollectionModalProps } from './useModalCreateCollection';
import useModalRenameCollection from './useModalRenameCollection';
import useModalViewFilters from './useModalViewFilters';
import css from './useTrialCollections.module.scss';

export interface TrialsCollectionInterface {
  controls: JSX.Element;
  filters: TrialFilters;
  openCreateModal: (p: CollectionModalProps) => void;
  setFilters: SetFilters;
  sorter: TrialSorter;
}

const collectionStoragePath = (projectId: string) => `collection/${projectId}`;

const configForProject = (projectId: string): SettingsConfig<{ collection: string }> => ({
  settings: {
    collection: {
      defaultValue: '',
      storageKey: 'collection',
      type: string,
    },
  },
  storagePath: collectionStoragePath(projectId),
});

const comparableStringification = (filters?: TrialFilters, sorter?: TrialSorter): string =>
  JSON.stringify([...Object.entries(filters ?? {}), ...Object.entries(sorter ?? {})].sort());

const defaultRanker = {
  rank: '0',
  sorter: { sortDesc: false, sortKey: 'searcherMetricValue' },
};

const getDefaultFilters = (projectId: string) => ({
  experimentIds: [],
  hparams: {},
  projectIds: [String(projectId)],
  ranker: clone(defaultRanker),
  searcher: '',
  states: [],
  tags: [],
  trainingMetrics: {},
  trialIds: [],
  userIds: [],
  validationMetrics: {},
  workspaceIds: [],
});

const defaultSorter: TrialSorter = {
  sortDesc: true,
  sortKey: 'trialId',
};

const MenuKey = {
  ClearFilters: 'Clear Filters',
  DeleteCollection: 'Delete Collection',
  RenameCollection: 'Rename Collection',
} as const;

export const useTrialCollections = (
  projectId: string,
  tableSettingsHook: UseSettingsReturn<InteractiveTableSettings>,
): TrialsCollectionInterface => {
  const { settings: tableSettings, updateSettings: updateTableSettings } = tableSettingsHook;
  const filterStorage = useStorage(`trial-filters}/${projectId ?? 1}`);
  const initFilters = filterStorage.getWithDefault<TrialFilters>(
    'filters',
    getDefaultFilters(projectId),
  );

  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));

  const userId = useMemo(() => (currentUser?.id ? String(currentUser?.id) : ''), [currentUser?.id]);

  const [
    // eslint-disable-next-line array-element-newline
    filters, // external filters
    _setFilters, // only use thru below wrapper
  ] = useState<TrialFilters>(initFilters);

  const setFilters = useCallback(
    (fs: FilterSetter) => {
      _setFilters((filters) => {
        if (!filters) return filters;
        const f = fs(filters);
        filterStorage.set('filters', f);
        return f;
      });
    },
    [filterStorage],
  );

  const sorter: TrialSorter = useMemo(
    () => ({
      ...defaultSorter,
      sortDesc: tableSettings.sortDesc,
      sortKey: tableSettings.sortKey ? String(tableSettings.sortKey) : '',
    }),
    [tableSettings.sortDesc, tableSettings.sortKey],
  );

  const filtersStringified = useMemo(
    () => comparableStringification(filters, sorter),
    [filters, sorter],
  );

  const [collectionFiltersStringified, setCollectionFiltersStringified] = useState<
    string | undefined
  >();

  const hasUnsavedFilters = useMemo(() => {
    if (!collectionFiltersStringified) return false;
    return filtersStringified !== collectionFiltersStringified;
  }, [collectionFiltersStringified, filtersStringified]);

  const [collections, setCollections] = useState<TrialsCollection[]>([]);

  const settingsConfig = useMemo(() => configForProject(projectId), [projectId]);

  const { settings, updateSettings } = useSettings<{ collection: string }>(settingsConfig);

  const previousCollectionStorage = useStorage(`previous-collection/${projectId}`);

  const getPreviousCollection = useCallback(
    () => previousCollectionStorage.get('collection'),
    [previousCollectionStorage],
  );

  const setPreviousCollection = useCallback(
    (c: TrialsCollection) => previousCollectionStorage.set('collection', c),
    [previousCollectionStorage],
  );

  const activeCollection = useMemo(
    () => collections.find((c) => c.name === settings.collection),
    [collections, settings.collection],
  );

  const fetchCollections = useCallback(async () => {
    const id = parseInt(projectId);
    if (isFiniteNumber(id)) {
      const response = await getTrialsCollections(id);
      const collections =
        response.collections
          ?.map(decodeTrialsCollection)
          .sort((a, b) => Number(b.userId === userId) - Number(a.userId === userId)) ?? [];
      setCollections(collections);
      return collections;
    }
  }, [projectId, userId]);

  useEffect(() => {
    fetchCollections();
  }, [fetchCollections]);

  const setCollection = useCallback(
    async (targetCollectionName: string, refetchBefore?: boolean) => {
      let _collections = collections;
      if (targetCollectionName) {
        if (refetchBefore) _collections = (await fetchCollections()) ?? _collections;
        const targetCollection = _collections.find((c) => c.name === targetCollectionName);
        if (targetCollection) {
          updateSettings({ collection: targetCollection.name });
        } else {
          updateSettings({ collection: undefined });
        }
      } else {
        _collections = (await fetchCollections()) ?? [];
        updateSettings({ collection: _collections?.[0]?.name });
      }
    },
    [collections, fetchCollections, updateSettings],
  );

  const saveCollection = useCallback(async () => {
    const newCollection = { ...activeCollection, filters, sorter } as TrialsCollection;
    await patchTrialsCollection(encodeTrialsCollection(newCollection));
    fetchCollections();
  }, [filters, activeCollection, sorter, fetchCollections]);

  const deleteCollection = useCallback(async () => {
    try {
      const id = finiteElseUndefined(activeCollection?.id);
      if (id !== undefined) {
        await deleteTrialsCollection(id);
      }
      await setCollection('', true);
    } catch (e) {
      handleError(e, {
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete collection.',
        silent: false,
        type: ErrorType.Api,
      });
    }
  }, [activeCollection, setCollection]);

  useEffect(() => {
    const previousCollection = getPreviousCollection();
    setCollectionFiltersStringified(
      comparableStringification(activeCollection?.filters, activeCollection?.sorter),
    );

    if (
      activeCollection &&
      JSON.stringify(activeCollection) !== JSON.stringify(previousCollection)
    ) {
      setFilters(() => activeCollection.filters);
      updateTableSettings({
        sortDesc: activeCollection.sorter.sortDesc,
        sortKey: activeCollection.sorter.sortKey,
      });
      setPreviousCollection(activeCollection);
    }
  }, [
    activeCollection,
    getPreviousCollection,
    setPreviousCollection,
    updateTableSettings,
    setFilters,
  ]);

  const userOwnsCollection = useMemo(() => {
    if (currentUser?.isAdmin) return true;

    return activeCollection?.userId === userId;
  }, [activeCollection?.userId, currentUser?.isAdmin, userId]);

  const handleAfterCreate = useCallback(
    async (collectionName: string) => {
      await setCollection(collectionName, true);
    },
    [setCollection],
  );

  const { modalOpen, contextHolder: collectionContextHolder } = useModalTrialCollection({
    onConfirm: handleAfterCreate,
    projectId,
  });

  const createCollectionFromFilters = useCallback(() => {
    modalOpen({ trials: { filters, sorter } });
  }, [filters, modalOpen, sorter]);

  const resetFiltersToCollection = useCallback(() => {
    if (activeCollection?.filters) setFilters(() => activeCollection?.filters);
    if (activeCollection?.sorter) updateTableSettings({ ...activeCollection?.sorter });
  }, [activeCollection, updateTableSettings, setFilters]);

  const clearFilters = useCallback(() => {
    setFilters(() => getDefaultFilters(projectId));
  }, [projectId, setFilters]);

  const { modalOpen: openFiltersModal, contextHolder: viewFiltersContextHolder } =
    useModalViewFilters();

  const viewFilters = useCallback(() => {
    openFiltersModal({ filters, sorter });
  }, [filters, openFiltersModal, sorter]);

  const handleRenameComplete = useCallback(
    async (name: string) => {
      await fetchCollections();
      updateSettings({ collection: name });
    },
    [fetchCollections, updateSettings],
  );

  const { modalOpen: openRenameModal, contextHolder: renameContextHolder } =
    useModalRenameCollection({ onComplete: handleRenameComplete });

  const renameCollection = useCallback(() => {
    const id = collections.find((c) => c.name === settings.collection)?.id;
    if (id) openRenameModal({ id, name: settings.collection ?? '' });
  }, [collections, settings.collection, openRenameModal]);

  const collectionIsActive = !!(collections.length && settings.collection);

  const menu = useMemo(
    () =>
      collectionIsActive
        ? [
            {
              disabled: !userOwnsCollection,
              key: MenuKey.RenameCollection,
              label: MenuKey.RenameCollection,
            },
            {
              disabled: !userOwnsCollection,
              key: MenuKey.DeleteCollection,
              label: MenuKey.DeleteCollection,
            },
            {
              key: MenuKey.ClearFilters,
              label: MenuKey.ClearFilters,
            },
          ]
        : [
            {
              key: MenuKey.ClearFilters,
              label: MenuKey.ClearFilters,
            },
          ],
    [collectionIsActive, userOwnsCollection],
  );

  const handleDropdown = useCallback(
    (key: string) => {
      switch (key) {
        case MenuKey.ClearFilters:
          clearFilters();
          break;
        case MenuKey.DeleteCollection:
          deleteCollection();
          break;
        case MenuKey.RenameCollection:
          renameCollection();
          break;
      }
    },
    [clearFilters, deleteCollection, renameCollection],
  );

  const controls = useMemo(
    () => (
      <div className={css.base}>
        <div className={css.options}>
          <Button onClick={createCollectionFromFilters}>New Collection</Button>
          <Select
            disabled={!collections.length}
            placeholder={collections?.length ? 'Select Collection' : 'No collections created'}
            status={settings.collection && hasUnsavedFilters ? 'warning' : undefined}
            style={{ width: '200px' }}
            value={collectionIsActive ? settings.collection : undefined}
            onChange={async (value) => await setCollection(value)}>
            {[
              ...(collections?.map((collection) => (
                <Select.Option key={collection.name} value={collection.name}>
                  {userId === collection.userId ? <Icon name="user-small" title="User" /> : '   '}{' '}
                  {collection.name}
                </Select.Option>
              )) ?? []),
            ]}
          </Select>
          <Button
            icon={<Icon name="settings" showTooltip title="View Active Filters" />}
            onClick={viewFilters}
          />
          <Button
            disabled={!userOwnsCollection || !collectionIsActive}
            icon={
              <Icon
                name="checkmark"
                showTooltip
                title={collectionIsActive ? 'Save Collection' : 'No Collection Active'}
              />
            }
            onClick={saveCollection}
          />
          <Button
            disabled={!collectionIsActive}
            icon={
              <Icon
                name="reset"
                showTooltip
                title={collectionIsActive ? 'Reset Filters to Collection' : 'No Collection Active'}
              />
            }
            onClick={resetFiltersToCollection}
          />
          <Dropdown menu={menu} onClick={handleDropdown}>
            <Button icon={<Icon name="overflow-vertical" title="Action menu" />} />
          </Dropdown>
          {viewFiltersContextHolder}
          {collectionContextHolder}
          {renameContextHolder}
        </div>
      </div>
    ),
    [
      collectionContextHolder,
      collectionIsActive,
      collections,
      createCollectionFromFilters,
      handleDropdown,
      hasUnsavedFilters,
      menu,
      renameContextHolder,
      resetFiltersToCollection,
      saveCollection,
      setCollection,
      settings.collection,
      userId,
      userOwnsCollection,
      viewFilters,
      viewFiltersContextHolder,
    ],
  );

  return {
    controls,
    filters,
    openCreateModal: modalOpen,
    setFilters,
    sorter,
  };
};
