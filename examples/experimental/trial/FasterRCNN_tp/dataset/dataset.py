# -*- coding: utf-8 -*-
__all__ = ["DatasetRegistry", "DatasetSplit"]


class DatasetSplit:
    """
    A class to load datasets, evaluate results for a datast split (e.g., "coco_train_2017")

    To use your own dataset that's not in COCO format, write a subclass that
    implements the interfaces.
    """

    def training_roidbs(self):
        """
        Returns:
            roidbs (list[dict]):

        Produce "roidbs" as a list of dict, each dict corresponds to one image with k>=0 instances.
        and the following keys are expected for training:

        file_name: str, full path to the image
        boxes: numpy array of kx4 floats, each row is [x1, y1, x2, y2]
        class: numpy array of k integers, in the range of [1, #categories], NOT [0, #categories)
        is_crowd: k booleans. Use k False if you don't know what it means.
        segmentation: k lists of numpy arrays (one for each instance).
            Each list of numpy arrays corresponds to the mask for one instance.
            Each numpy array in the list is a polygon of shape Nx2,
            because one mask can be represented by N polygons.

            If your segmentation annotations are originally masks rather than polygons,
            either convert it, or the augmentation will need to be changed or skipped accordingly.

            Include this field only if training Mask R-CNN.
        """
        raise NotImplementedError()

    def inference_roidbs(self):
        """
        Returns:
            roidbs (list[dict]):

            Each dict corresponds to one image to run inference on. The
            following keys in the dict are expected:

            file_name (str): full path to the image
            image_id (str): an id for the image. The inference results will be stored with this id.
        """
        raise NotImplementedError()

    def eval_inference_results(self, results, output=None):
        """
        Args:
            results (list[dict]): the inference results as dicts.
                Each dict corresponds to one __instance__. It contains the following keys:

                image_id (str): the id that matches `inference_roidbs`.
                category_id (int): the category prediction, in range [1, #category]
                bbox (list[float]): x1, y1, x2, y2
                score (float):
                segmentation: the segmentation mask in COCO's rle format.
            output (str): the output file or directory to optionally save the results to.

        Returns:
            dict: the evaluation results.
        """
        raise NotImplementedError()


class DatasetRegistry:
    _registry = {}

    @staticmethod
    def register(name, func):
        """
        Args:
            name (str): the name of the dataset split, e.g. "coco_train2017"
            func: a function which returns an instance of `DatasetSplit`
        """
        assert name not in DatasetRegistry._registry, "Dataset {} was registered already!".format(
            name
        )
        DatasetRegistry._registry[name] = func

    @staticmethod
    def get(name):
        """
        Args:
            name (str): the name of the dataset split, e.g. "coco_train2017"

        Returns:
            DatasetSplit
        """
        assert name in DatasetRegistry._registry, "Dataset {} was not registered!".format(name)
        return DatasetRegistry._registry[name]()
