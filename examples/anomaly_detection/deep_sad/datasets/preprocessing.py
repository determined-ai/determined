import torch
import numpy as np


def create_semisupervised_setting(labels, normal_classes, outlier_classes, known_outlier_classes,
                                  ratio_known_normal, ratio_known_outlier, ratio_pollution):
    """
    Create a semi-supervised data setting.
    :param labels: np.array with labels of all dataset samples
    :param normal_classes: tuple with normal class labels
    :param outlier_classes: tuple with anomaly class labels
    :param known_outlier_classes: tuple with known (labeled) anomaly class labels
    :param ratio_known_normal: the desired ratio of known (labeled) normal samples
    :param ratio_known_outlier: the desired ratio of known (labeled) anomalous samples
    :param ratio_pollution: the desired pollution ratio of the unlabeled data with unknown (unlabeled) anomalies.
    :return: tuple with list of sample indices, list of original labels, and list of semi-supervised labels
    """
    idx_normal = np.argwhere(np.isin(labels, normal_classes)).flatten()
    idx_outlier = np.argwhere(np.isin(labels, outlier_classes)).flatten()
    idx_known_outlier_candidates = np.argwhere(np.isin(labels, known_outlier_classes)).flatten()

    n_normal = len(idx_normal)

    # Solve system of linear equations to obtain respective number of samples
    a = np.array([[1, 1, 0, 0],
                  [(1-ratio_known_normal), -ratio_known_normal, -ratio_known_normal, -ratio_known_normal],
                  [-ratio_known_outlier, -ratio_known_outlier, -ratio_known_outlier, (1-ratio_known_outlier)],
                  [0, -ratio_pollution, (1-ratio_pollution), 0]])
    b = np.array([n_normal, 0, 0, 0])
    x = np.linalg.solve(a, b)

    # Get number of samples
    n_known_normal = int(x[0])
    n_unlabeled_normal = int(x[1])
    n_unlabeled_outlier = int(x[2])
    n_known_outlier = int(x[3])

    # Sample indices
    perm_normal = np.random.permutation(n_normal)
    perm_outlier = np.random.permutation(len(idx_outlier))
    perm_known_outlier = np.random.permutation(len(idx_known_outlier_candidates))

    idx_known_normal = idx_normal[perm_normal[:n_known_normal]].tolist()
    idx_unlabeled_normal = idx_normal[perm_normal[n_known_normal:n_known_normal+n_unlabeled_normal]].tolist()
    idx_unlabeled_outlier = idx_outlier[perm_outlier[:n_unlabeled_outlier]].tolist()
    idx_known_outlier = idx_known_outlier_candidates[perm_known_outlier[:n_known_outlier]].tolist()

    # Get original class labels
    labels_known_normal = labels[idx_known_normal].tolist()
    labels_unlabeled_normal = labels[idx_unlabeled_normal].tolist()
    labels_unlabeled_outlier = labels[idx_unlabeled_outlier].tolist()
    labels_known_outlier = labels[idx_known_outlier].tolist()

    # Get semi-supervised setting labels
    semi_labels_known_normal = np.ones(n_known_normal).astype(np.int32).tolist()
    semi_labels_unlabeled_normal = np.zeros(n_unlabeled_normal).astype(np.int32).tolist()
    semi_labels_unlabeled_outlier = np.zeros(n_unlabeled_outlier).astype(np.int32).tolist()
    semi_labels_known_outlier = (-np.ones(n_known_outlier).astype(np.int32)).tolist()

    # Create final lists
    list_idx = idx_known_normal + idx_unlabeled_normal + idx_unlabeled_outlier + idx_known_outlier
    list_labels = labels_known_normal + labels_unlabeled_normal + labels_unlabeled_outlier + labels_known_outlier
    list_semi_labels = (semi_labels_known_normal + semi_labels_unlabeled_normal + semi_labels_unlabeled_outlier
                        + semi_labels_known_outlier)

    return list_idx, list_labels, list_semi_labels
