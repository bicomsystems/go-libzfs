/* C wrappers around some zfs calls and C in general that should simplify
 * using libzfs from go language, and make go code shorter and more readable.
 */
 
#include <libzfs.h>
#include <memory.h>
#include <string.h>
#include <stdio.h>

#include "zpool.h"

static char _lasterr_[1024];

const char *lasterr(void) {
	return _lasterr_;
}

zpool_list_t *create_zpool_list_item() {
	zpool_list_t *zlist = malloc(sizeof(zpool_list_t));
	memset(zlist, 0, sizeof(zpool_list_t));
	return zlist;
}

int zpool_list_callb(zpool_handle_t *pool, void *data) {
	zpool_list_t **lroot = (zpool_list_t**)data;
	zpool_list_t *nroot = create_zpool_list_item();

	if ( !((*lroot)->zph) ) {
		(*lroot)->zph = pool;
	} else {
		nroot->zph = pool;
		nroot->pnext = (void*)*lroot;
		*lroot = nroot;
	}
	return 0;
}

int zpool_list(libzfs_handle_t *libzfs, zpool_list_t **first) {
	int err = 0;
	zpool_list_t *zlist = create_zpool_list_item();
	err = zpool_iter(libzfs, zpool_list_callb, &zlist);
	if ( zlist->zph ) {
		*first = zlist;
	} else {
		*first = 0;
		free(zlist);
	}
	return err;
}

zpool_list_t* zpool_list_open(libzfs_handle_t *libzfs, const char *name) {
	zpool_list_t *zlist = create_zpool_list_item();
	zlist->zph = zpool_open(libzfs, name);
	if ( zlist->zph ) {
		return zlist;
	} else {
		free(zlist);
	}
	return 0;
}

zpool_list_t *zpool_next(zpool_list_t *pool) {
	return pool->pnext;
}

void zpool_list_close(zpool_list_t *pool) {
	zpool_close(pool->zph);
	free(pool);
}

property_list_t *new_property_list() {
	property_list_t *r = malloc(sizeof(property_list_t));
	memset(r, 0, sizeof(property_list_t));
	return r;
}

void free_properties(property_list_t *root) {
	if (root != 0) {
		property_list_t *tmp = 0;
		do {
			tmp = root->pnext;
			free(root);
			root = tmp;
		} while(tmp);
	}
}

property_list_t *next_property(property_list_t *list) {
	if (list != 0) {
		return list->pnext;
	}
	return list;
}


void zprop_source_tostr(char *dst, zprop_source_t source) {
	switch (source) {
	case ZPROP_SRC_NONE:
		strcpy(dst, "none");
		break;
	case ZPROP_SRC_TEMPORARY:
		strcpy(dst, "temporary");
		break;
	case ZPROP_SRC_LOCAL:
		strcpy(dst, "local");
		break;
	case ZPROP_SRC_INHERITED:
		strcpy(dst, "inherited");
		break;
	case ZPROP_SRC_RECEIVED:
		strcpy(dst, "received");
		break;
	default:
		strcpy(dst, "default");
		break; 
	}
}


int read_zpool_property(zpool_handle_t *zh, property_list_t *list, int prop) {
	
	int r = 0;
	zprop_source_t source;

	r = zpool_get_prop(zh, prop, 
		list->value, INT_MAX_VALUE, &source);
	if (r == 0) {
		// strcpy(list->name, zpool_prop_to_name(prop));
		zprop_source_tostr(list->source, source);
	}
	list->property = (int)prop;
	return r;
}

int read_append_zpool_property(zpool_handle_t *zh, property_list_t **proot,
	zpool_prop_t prop) {
	int r = 0;
	property_list_t *newitem = NULL, *root = *proot;
	newitem = new_property_list();

	r = read_zpool_property(zh, newitem, prop);
	// printf("p: %s %s %s\n", newitem->name, newitem->value, newitem->source);
	newitem->pnext = root;
	*proot = root = newitem;
	if (r != 0) {
		free_properties(root);
		*proot = NULL;
	}
	return r;
}

property_list_t *read_zpool_properties(zpool_handle_t *zh) {
	// read pool name as first property
	property_list_t *root = NULL, *list = NULL;

	int r = read_append_zpool_property(zh, &root, ZPOOL_PROP_NAME);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_SIZE);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_CAPACITY);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_ALTROOT);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_HEALTH);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_GUID);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_VERSION);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_BOOTFS);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_DELEGATION);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_AUTOREPLACE);
	if (r != 0) {
		return 0;
	}


	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_CACHEFILE);
	if (r != 0) {
		return 0;
	}


	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_FAILUREMODE);
	if (r != 0) {
		return 0;
	}


	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_LISTSNAPS);
	if (r != 0) {
		return 0;
	}


	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_AUTOEXPAND);
	if (r != 0) {
		return 0;
	}


	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_DEDUPDITTO);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_DEDUPRATIO);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_FREE);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_ALLOCATED);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_READONLY);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_ASHIFT);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_COMMENT);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_EXPANDSZ);
	if (r != 0) {
		return 0;
	}

	r = read_append_zpool_property(zh, &root, ZPOOL_PROP_FREEING);
	if (r != 0) {
		return 0;
	}


	list = new_property_list();
	list->property = ZPOOL_NUM_PROPS;
	sprintf(list->value, "%d", ZPOOL_NUM_PROPS);
	list->pnext = root;
	zprop_source_tostr(list->source, ZPROP_SRC_NONE);
	root = list;

	// printf("Finished properties reading.\n");
	return root;
}

pool_state_t zpool_read_state(zpool_handle_t *zh) {
	return zpool_get_state(zh);
}


const char *gettext(const char *txt) {
	return txt;
}
/*
 * Add a property pair (name, string-value) into a property nvlist.
 */
int
add_prop_list(const char *propname, char *propval, nvlist_t **props,
	boolean_t poolprop) {
	zpool_prop_t prop = ZPROP_INVAL;
	zfs_prop_t fprop;
	nvlist_t *proplist;
	const char *normnm;
	char *strval;

	if (*props == NULL &&
	    nvlist_alloc(props, NV_UNIQUE_NAME, 0) != 0) {
		(void) snprintf(_lasterr_, 1024, "internal error: out of memory");
		return (1);
	}

	proplist = *props;

	if (poolprop) {
		const char *vname = zpool_prop_to_name(ZPOOL_PROP_VERSION);

		if ((prop = zpool_name_to_prop(propname)) == ZPROP_INVAL &&
		    !zpool_prop_feature(propname)) {
			(void) snprintf(_lasterr_, 1024, "property '%s' is "
			    "not a valid pool property", propname);
			return (2);
		}

		/*
		 * feature@ properties and version should not be specified
		 * at the same time.
		 */
		// if ((prop == ZPROP_INVAL && zpool_prop_feature(propname) &&
		//     nvlist_exists(proplist, vname)) ||
		//     (prop == ZPOOL_PROP_VERSION &&
		//     prop_list_contains_feature(proplist))) {
		// 	(void) fprintf(stderr, gettext("'feature@' and "
		// 	    "'version' properties cannot be specified "
		// 	    "together\n"));
		// 	return (2);
		// }


		if (zpool_prop_feature(propname))
			normnm = propname;
		else
			normnm = zpool_prop_to_name(prop);
	} else {
		if ((fprop = zfs_name_to_prop(propname)) != ZPROP_INVAL) {
			normnm = zfs_prop_to_name(fprop);
		} else {
			normnm = propname;
		}
	}

	if (nvlist_lookup_string(proplist, normnm, &strval) == 0 &&
	    prop != ZPOOL_PROP_CACHEFILE) {
		(void) snprintf(_lasterr_, 1024, "property '%s' "
		    "specified multiple times", propname);
		return (2);
	}

	if (nvlist_add_string(proplist, normnm, propval) != 0) {
		(void) snprintf(_lasterr_, 1024, "internal "
		    "error: out of memory\n");
		return (1);
	}

	return (0);
}

nvlist_t** nvlist_alloc_array(int count) {
	return malloc(count*sizeof(nvlist_t*));
}

void nvlist_array_set(nvlist_t** a, int i, nvlist_t *item) {
	a[i] = item;
}

void nvlist_free_array(nvlist_t **a) {
	free(a);
}

void free_cstring(char *str) {
	free(str);
}
