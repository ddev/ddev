/**
 * ddev-user.nsi - DDEV Local Setup Script for the User Installer
 */

/**
 * Include Pre Setup Header
 */
!include "ddev-setup.nsh"

/**
 * Add local include and plugin directories
 */
!define DDEV_INSTALLER_TYPE ${DDEV_INSTALLER_TYPE_USER}

/**
 * Include Main Setup Header
 */
!include "ddev-setup.nsh"
