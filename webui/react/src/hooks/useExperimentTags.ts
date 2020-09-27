import { useCallback } from 'react';

import { patchExperiment } from 'services/api';

export interface UseExperimentTagsInterface {
  handleTagListChange: (id: number) => (oldTag: string, newTag: string) => void
  handleTagListCreate: (id: number) => (tag: string) => void;
  handleTagListDelete: (id: number) => (tag: string) => void;
}

const useExperimentTags = (callbackFn?: (() => void)): UseExperimentTagsInterface => {
  const updateTags = useCallback(async (id: number, labels: Record<string, boolean | null>) => {
    await patchExperiment({ body: { labels }, experimentId: id });
    if (typeof callbackFn === 'function') {
      callbackFn();
    }
  }, [ callbackFn ]);

  return {
    handleTagListChange: useCallback((id: number) => (oldTag: string, newTag: string) => {
      updateTags(id, { [newTag]: true, [oldTag]: null });
    }, [ updateTags ]),
    handleTagListCreate: useCallback((id: number) => (tag: string) => {
      updateTags(id, { [tag]: true });
    }, [ updateTags ]),
    handleTagListDelete: useCallback((id: number) => (tag: string) => {
      updateTags(id, { [tag]: null });
    }, [ updateTags ]),
  };
};

export default useExperimentTags;
