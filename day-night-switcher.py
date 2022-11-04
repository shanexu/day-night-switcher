#!/usr/bin/python

from datetime import datetime, timedelta
import time
import math
import logging
import os


logging.basicConfig(level=logging.INFO, format='%(asctime)s %(message)s')


def sleep(seconds: int):
    logging.info('sleep %d seconds', seconds)
    time.sleep(seconds)


def ceil_seconds(start: datetime, stop: datetime) -> int:
    return math.ceil((stop - start).total_seconds())


def day():
    logging.info('day')
    os.system('/home/shane/bin/night-theme-switch.sh')


def night():
    logging.info('night')
    os.system('/home/shane/bin/night-theme-switch.sh')


while True:
    time.sleep(10)
    now = datetime.now()
    day_begin = datetime(now.year, month=now.month, day=now.day, hour=6)
    night_begin = datetime(now.year, month=now.month, day=now.day, hour=18)

    if now < day_begin:
        night()
        sleep(ceil_seconds(now, day_begin))
    elif now < night_begin:
        day()
        sleep(ceil_seconds(now, night_begin))
    else:
        night()
        sleep(ceil_seconds(now, day_begin + timedelta(days=1)))
