# -*- Makefile -*-
#
# Copyright 2006 Sandia Corporation. Under the terms of Contract
# DE-AC04-94AL85000 with Sandia Corporation, the U.S. Government
# retains certain rights in this software.
#
# This program is free software; you can redistribute it and/or
# modify it under the terms of the GNU General Public License
# as published by the Free Software Foundation; either version 2
# of the License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program; if not, write to the Free Software
# Foundation, Inc., 51 Franklin Street, Fifth Floor, 
# Boston, MA  02110-1301, USA.

RELEASE_VERSION = 1.0

CC = mpicc
CPPFLAGS = -DVERSION=$(RELEASE_VERSION)
CFLAGS = -O3

msgrate: msgrate.o

clean:
	rm -f msgrate.o msgrate *~

dist:
	rm -rf msgrate-$(RELEASE_VERSION)
	mkdir msgrate-$(RELEASE_VERSION)
	(for file in msgrate.c README Makefile GPL.txt LICENSE.txt ; do \
	cp $$file msgrate-$(RELEASE_VERSION)/. ; \
	chmod 644 msgrate-$(RELEASE_VERSION)/$$file ; \
	done)
	tar czf msgrate-$(RELEASE_VERSION).tar.gz msgrate-$(RELEASE_VERSION)
	rm -rf msgrate-$(RELEASE_VERSION)
