import React, { ReactNode, useEffect, useState } from 'react';

import Tags, { TagAction } from 'components/kit/Tags';
import { updateTrialTags } from 'services/api';
import { V1AugmentedTrial } from 'services/api-ts-sdk';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';

export const addTagFunc =
  (trialId: number) =>
  async (tag: string): Promise<unknown> =>
    await updateTrialTags({
      patch: { addTag: [{ key: tag }] },
      trial: { ids: [trialId] },
    });

export const removeTagFunc =
  (trialId: number) =>
  async (tag: string): Promise<unknown> =>
    await updateTrialTags({
      patch: { removeTag: [{ key: tag }] },
      trial: { ids: [trialId] },
    });

interface Props {
  onAdd: (tag: string) => Promise<unknown>;
  onRemove: (tag: string) => Promise<unknown>;
  tags: string[];
}

const TagList: React.FC<Props> = ({ tags: _tags, onAdd, onRemove }) => {
  const [tags, setTags] = useState(_tags);

  useEffect(() => setTags(_tags), [_tags]);

  const handleTagAction = async (action: TagAction, tag: string) => {
    try {
      if (action === TagAction.Add) {
        await onAdd(tag);
        setTags([...tags.filter((t) => t !== tag), tag]);
      } else if (action === TagAction.Remove) {
        await onRemove(tag);
        setTags((tags) => tags.filter((t) => t !== tag));
      }
    } catch (e) {
      handleError(e, {
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to add tag.',
        silent: false,
        type: ErrorType.Api,
      });
    }
  };

  return <Tags compact tags={tags} onAction={handleTagAction} />;
};

const trialTagsRenderer = (value: string, record: V1AugmentedTrial): ReactNode => (
  <TagList
    tags={Object.keys(record.tags)}
    onAdd={addTagFunc(record.trialId)}
    onRemove={removeTagFunc(record.trialId)}
  />
);

export default trialTagsRenderer;
