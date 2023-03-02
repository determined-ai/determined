import { GetMasterResponseProduct } from 'services/api-ts-sdk';
import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import { Loadable } from 'utils/loadable';

interface ProductHook {
  isCommunity: boolean;
}

const useProduct = (): ProductHook => {
  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());
  return { isCommunity: info.product === GetMasterResponseProduct.COMMUNITY };
};

export default useProduct;
