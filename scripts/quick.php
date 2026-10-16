<?php

function Quick(Array $arg):Array  {
    $l = count( $arg);
    if ($l<=1) {
        return $arg;
    }
    $poivt= $arg[0];
     $left =array();
     $right = array();
    for ($i=1;$i<$l;$i++){ 
         if ($poivt>=$arg[$i]){
            $left[]=$arg[$i] ;
         }else{
            $right[]=$arg[$i];
         }
    }
   return  array_merge(Quick($left),[$poivt],Quick($right));

}
function quickSort($arr) {
    if (count($arr) <= 1) {
        return $arr;
    }
    $pivot = $arr[0];
    $left = $right = array();
    for ($i = 1; $i < count($arr); $i++) {
        if ($arr[$i] <= $pivot) {
            $left[] = $arr[$i];
        } else {
            $right[] = $arr[$i];
        }
    }
    return array_merge(quickSort($left), array($pivot), quickSort($right));
}
print_r(Quick([1,2,1,4,6,7]));