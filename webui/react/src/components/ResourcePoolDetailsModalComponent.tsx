import Divider from 'hew/Divider';
import { Modal } from 'hew/Modal';
import React, { Fragment, useCallback } from 'react';

import JsonGlossary from 'components/JsonGlossary';
import { V1ResourcePoolDetail } from 'services/api-ts-sdk';
import { JsonObject, ResourcePool } from 'types';
import { camelCaseToSentence } from 'utils/string';

import css from './ResourcePoolDetails.module.scss';

interface Props {
  onCloseModal?: () => void;
  resourcePool: ResourcePool;
}

const ResourcePoolDetailsModalComponent: React.FC<Props> = ({
  resourcePool: pool,
  onCloseModal,
}: Props) => {
  const { details, ...mainSection } = pool;

  const handleClose = useCallback(() => {
    if (onCloseModal) onCloseModal();
  }, [onCloseModal]);

  for (const key in details) {
    if (details[key as keyof V1ResourcePoolDetail] === null) {
      delete details[key as keyof V1ResourcePoolDetail];
    }
  }

  return (
    <Modal size="medium" title={pool.name} onClose={handleClose}>
      <JsonGlossary
        json={mainSection as unknown as JsonObject}
        translateLabel={camelCaseToSentence}
      />
      {Object.keys(details).map((key) => (
        <Fragment key={key}>
          <Divider />
          <div className={css.subTitle}>{camelCaseToSentence(key)}</div>
          <JsonGlossary
            json={details[key as keyof V1ResourcePoolDetail] as unknown as JsonObject}
            translateLabel={camelCaseToSentence}
          />
        </Fragment>
      ))}
    </Modal>
  );
};

export default ResourcePoolDetailsModalComponent;
