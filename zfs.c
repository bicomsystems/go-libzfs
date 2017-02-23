/* C wrappers around some zfs calls and C in general that should simplify
 * using libzfs from go language, make go code shorter and more readable.
 */

#include <libzfs.h>
#include <memory.h>
#include <string.h>
#include <stdio.h>

#include "zpool.h"
#include "zfs.h"


dataset_list_t *create_dataset_list_item() {
	dataset_list_t *zlist = malloc(sizeof(dataset_list_t));
	memset(zlist, 0, sizeof(dataset_list_t));
	return zlist;
}

void dataset_list_close(dataset_list_t *list) {
	zfs_close(list->zh);
	free(list);
	// dataset_list_free(list);
}

void dataset_list_free(dataset_list_t *list) {
	dataset_list_t *next;
	while(list) {
		next = list->pnext;
		free(list);
		list = next;
	}
}

int dataset_list_callb(zfs_handle_t *dataset, void *data) {
	dataset_list_t **lroot = (dataset_list_t**)data;
	dataset_list_t *nroot = create_dataset_list_item();

	if ( !((*lroot)->zh) ) {
		(*lroot)->zh = dataset;
	} else {
		nroot->zh = dataset;
		nroot->pnext = (void*)*lroot;
		*lroot = nroot;
	}
	return 0;
}

int dataset_list_root(libzfs_handle_t *libzfs, dataset_list_t **first) {
	int err = 0;
	dataset_list_t *zlist = create_dataset_list_item();
	err = zfs_iter_root(libzfs, dataset_list_callb, &zlist);
	if ( zlist->zh ) {
		*first = zlist;
	} else {
		*first = 0;
		dataset_list_free(zlist);
	}
	return err;
}

dataset_list_t *dataset_next(dataset_list_t *dataset) {
	return dataset->pnext;
}


int dataset_list_children(zfs_handle_t *zfs, dataset_list_t **first) {
	int err = 0;
	dataset_list_t *zlist = create_dataset_list_item();
	err = zfs_iter_children(zfs, dataset_list_callb, &zlist);
	if ( zlist->zh ) {
		*first = zlist;
	} else {
		*first = 0;
		dataset_list_free(zlist);
	}
	return err;
}

int read_dataset_property(zfs_handle_t *zh, property_list_t *list, int prop) {
	int r = 0;
	zprop_source_t source;
	char statbuf[INT_MAX_VALUE];

	r = zfs_prop_get(zh, prop,
		list->value, INT_MAX_VALUE, &source, statbuf, INT_MAX_VALUE, 1);
	if (r == 0) {
		// strcpy(list->name, zpool_prop_to_name(prop));
		zprop_source_tostr(list->source, source);
	}
	list->property = (int)prop;
	return r;
}

int read_user_property(zfs_handle_t *zh, property_list_t *list, const char *prop) {
	nvlist_t *user_props = zfs_get_user_props(zh);
	nvlist_t *propval;
	zprop_source_t sourcetype;
	char *strval;
	char *sourceval;
	// char source[ZFS_MAX_DATASET_NAME_LEN];
	
	if (nvlist_lookup_nvlist(user_props,
		prop, &propval) != 0) {
		sourcetype = ZPROP_SRC_NONE;
		(void) strncpy(list->source,
				"none", sizeof (list->source));
		strval = "-";
	} else {
		verify(nvlist_lookup_string(propval,
			ZPROP_VALUE, &strval) == 0);
		verify(nvlist_lookup_string(propval,
			ZPROP_SOURCE, &sourceval) == 0);

		if (strcmp(sourceval,
			zfs_get_name(zh)) == 0) {
			sourcetype = ZPROP_SRC_LOCAL;
			(void) strncpy(list->source,
				"local", sizeof (list->source));
		} else if (strcmp(sourceval,
			ZPROP_SOURCE_VAL_RECVD) == 0) {
			sourcetype = ZPROP_SRC_RECEIVED;
			(void) strncpy(list->source,
				"received", sizeof (list->source));
		} else {
			sourcetype = ZPROP_SRC_INHERITED;
			(void) strncpy(list->source,
				sourceval, sizeof (list->source));
		}
	}
	(void) strncpy(list->value,
				strval, sizeof (list->value));
	return 0;
}

int clear_last_error(libzfs_handle_t *hdl) {
	zfs_standard_error(hdl, EZFS_SUCCESS, "success");
	return 0;
}

char** alloc_cstrings(int size) {
	return malloc(size*sizeof(char*));
}

void strings_setat(char **a, int at, char *v) {
	a[at] = v;
}
